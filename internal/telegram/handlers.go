package telegram

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/sergey/GoogleTaskBot/internal/tasks"
	tele "gopkg.in/telebot.v3"
)

func (b *Bot) onStart(c tele.Context) error {
	return sendHTML(c,
		"<b>Google Tasks</b>\n\n"+
			"Управляйте задачами из Telegram:\n"+
			"• просмотр и фильтры (все / просроченные / сегодня / завтра)\n"+
			"• поиск: /find текст\n"+
			"• списки Google Tasks\n"+
			"• добавление, срок, заметки\n"+
			"• выполнить / вернуть / удалить\n\n"+
			"<i>Нажмите на задачу в списке, чтобы открыть карточку.</i>",
		mainMenu(),
	)
}

func (b *Bot) onHelp(c tele.Context) error {
	return sendHTML(c,
		"<b>Справка</b>\n\n"+
			"<b>Меню</b> — фильтры и списки.\n"+
			"<b>➕ Добавить</b> или <code>/add название</code> — новая задача.\n"+
			"<code>/find текст</code> — поиск по задачам.\n"+
			"В карточке: выполнить, срок, название, заметки, ссылка на Google.\n\n"+
			"🔴 — просрочено · 🟡 — сегодня · 🔵 — завтра · 🟢 — позже · ⚪ — без срока",
		mainMenu(),
	)
}

func (b *Bot) onAllTasks(c tele.Context) error {
	return b.loadAndShow(c, viewAll, "", "", 0)
}

func (b *Bot) onOverdue(c tele.Context) error {
	return b.loadAndShow(c, viewOverdue, "", "", 0)
}

func (b *Bot) onTodayTasks(c tele.Context) error {
	return b.loadAndShow(c, viewToday, "", "", 0)
}

func (b *Bot) onTomorrowTasks(c tele.Context) error {
	return b.loadAndShow(c, viewTomorrow, "", "", 0)
}

func (b *Bot) onFind(c tele.Context) error {
	query := strings.TrimSpace(strings.Join(c.Args(), " "))
	if query == "" {
		return sendHTML(c, "🔍 Введите: <code>/find текст</code>", mainMenu())
	}
	b.sessions.SetSearchQuery(c.Sender().ID, query)
	return b.loadAndShow(c, viewSearch, "", "", 0)
}

func (b *Bot) onListsMenu(c tele.Context) error {
	ctx := context.Background()
	lists, err := b.client.ListTaskLists(ctx)
	if err != nil {
		return sendHTML(c, "❌ "+htmlEsc(err.Error()), mainMenu())
	}
	b.sessions.SetPickerLists(c.Sender().ID, lists)
	return replyOrEdit(c, "<b>Выберите список</b>", listPickerButtons(lists))
}

func (b *Bot) onCompleted(c tele.Context) error {
	return b.loadAndShow(c, viewCompleted, "", "", 0)
}

func (b *Bot) onAddStart(c tele.Context) error {
	args := strings.TrimSpace(strings.Join(c.Args(), " "))
	if args != "" {
		b.sessions.StartAddWithTitle(c.Sender().ID, args)
		ctx := context.Background()
		lists, err := b.client.ListTaskLists(ctx)
		if err != nil {
			b.sessions.Reset(c.Sender().ID)
			return b.sendErr(c, err)
		}
		b.sessions.SetPickerLists(c.Sender().ID, lists)
		return sendHTML(c, "📂 <b>Выберите список</b>\n\n"+htmlEsc(args), listPickerButtons(lists))
	}
	b.sessions.StartAdd(c.Sender().ID)
	return sendHTML(c, "➕ <b>Новая задача</b>\n\nВведите название:", nil)
}

func (b *Bot) loadAndShow(c tele.Context, kind viewKind, listID, listName string, page int) error {
	ctx := context.Background()
	loc := b.cfg.Timezone
	var (
		items []tasks.Task
		title string
		err   error
	)

	switch kind {
	case viewAll:
		title = "Все активные задачи"
		items, err = b.client.ListAllActiveTasks(ctx)
	case viewOverdue:
		title = "Просроченные"
		all, e := b.client.ListAllActiveTasks(ctx)
		err = e
		items = tasks.FilterOverdue(all, loc)
	case viewToday:
		title = "На сегодня"
		all, e := b.client.ListAllActiveTasks(ctx)
		err = e
		items = tasks.FilterToday(all, loc)
	case viewTomorrow:
		title = "На завтра"
		all, e := b.client.ListAllActiveTasks(ctx)
		err = e
		items = tasks.FilterTomorrow(all, loc)
	case viewSearch:
		q := b.sessions.SearchQuery(c.Sender().ID)
		title = "Поиск: " + q
		all, e := b.client.ListAllActiveTasks(ctx)
		err = e
		items = tasks.FilterSearch(all, q)
	case viewList:
		title = listName
		raw, e := b.client.ListTasks(ctx, listID, false)
		err = e
		for i := range raw {
			if raw[i].Status != "completed" {
				raw[i].ListName = listName
				items = append(items, raw[i])
			}
		}
	case viewCompleted:
		title = "Выполненные"
		items, err = b.client.ListCompletedTasks(ctx, 30)
	default:
		return nil
	}
	if err != nil {
		return b.sendErr(c, err)
	}

	tasks.SortForDisplay(items, loc)
	return b.renderList(c, kind, listID, listName, title, items, page)
}

func (b *Bot) renderList(c tele.Context, kind viewKind, listID, listName, title string, items []tasks.Task, page int) error {
	loc := b.cfg.Timezone
	msg, totalPages := buildListMessage(title, items, loc, page)
	b.sessions.SetView(c.Sender().ID, &viewState{
		Kind: kind, ListID: listID, ListName: listName, Tasks: items, Page: page,
	})
	markup := taskListButtons(items, page, totalPages, kind)
	return replyOrEdit(c, msg, markup)
}

func (b *Bot) reloadCurrentView(c tele.Context) error {
	v := b.sessions.View(c.Sender().ID)
	if v == nil {
		return sendHTML(c, "🏠 Меню", mainMenu())
	}
	return b.loadAndShow(c, v.Kind, v.ListID, v.ListName, v.Page)
}

func (b *Bot) openTask(c tele.Context, index int) error {
	t, ok := b.sessions.TaskAt(c.Sender().ID, index)
	if !ok {
		return c.Respond(&tele.CallbackResponse{Text: "Обновите список"})
	}
	b.sessions.SetTaskIndex(c.Sender().ID, index)

	msg := buildTaskDetail(t, b.cfg.Timezone)
	return replyOrEdit(c, msg, taskDetailButtons(t, strconv.Itoa(index)))
}

func (b *Bot) openTaskByRef(c tele.Context, listID, taskID string) error {
	ctx := context.Background()
	t, err := b.client.GetTask(ctx, listID, taskID)
	if err != nil {
		return b.sendErr(c, err)
	}
	lists, _ := b.client.ListTaskLists(ctx)
	for _, l := range lists {
		if l.ID == listID {
			t.ListName = l.Title
			break
		}
	}
	b.sessions.SetView(c.Sender().ID, &viewState{
		Kind: viewAll, Tasks: []tasks.Task{*t}, Page: 0,
	})
	b.sessions.SetTaskIndex(c.Sender().ID, 0)
	msg := buildTaskDetail(*t, b.cfg.Timezone)
	return sendHTML(c, msg, taskDetailButtons(*t, "0"))
}

func (b *Bot) backToList(c tele.Context) error {
	return b.reloadCurrentView(c)
}

func listTitle(v *viewState) string {
	switch v.Kind {
	case viewAll:
		return "Все активные задачи"
	case viewOverdue:
		return "Просроченные"
	case viewToday:
		return "На сегодня"
	case viewTomorrow:
		return "На завтра"
	case viewSearch:
		return "Поиск"
	case viewCompleted:
		return "Выполненные"
	case viewList:
		return v.ListName
	default:
		return "Задачи"
	}
}

func (b *Bot) onText(c tele.Context) error {
	userID := c.Sender().ID
	text := strings.TrimSpace(c.Text())
	if text == "" {
		return nil
	}

	action, taskIdx := b.sessions.PendingTaskIndex(userID)
	ctx := context.Background()

	switch action {
	case pendingTitle:
		b.sessions.SetTitle(userID, text)
		lists, err := b.client.ListTaskLists(ctx)
		if err != nil {
			b.sessions.Reset(userID)
			return b.sendErr(c, err)
		}
		b.sessions.SetPickerLists(userID, lists)
		return sendHTML(c, "📂 <b>Выберите список</b>", listPickerButtons(lists))

	case pendingNotesCreate:
		return b.finishCreateTask(c, text)

	case pendingNotes:
		t, ok := b.sessions.TaskAt(userID, taskIdx)
		if !ok {
			b.sessions.Reset(userID)
			return sendHTML(c, "Сессия устарела. Откройте задачу снова.", mainMenu())
		}
		if err := b.client.UpdateNotes(ctx, t.ListID, t.ID, text); err != nil {
			return b.sendErr(c, err)
		}
		b.sessions.ClearPending(userID)
		updated, err := b.client.GetTask(ctx, t.ListID, t.ID)
		if err != nil {
			return sendHTML(c, "📝 Заметки сохранены", mainMenu())
		}
		updated.ListName = t.ListName
		b.sessions.UpdateTaskInView(userID, *updated)
		return sendHTML(c, "📝 <b>Заметки сохранены</b>\n\n"+buildTaskDetail(*updated, b.cfg.Timezone), taskDetailButtons(*updated, strconv.Itoa(taskIdx)))

	case pendingRename:
		t, ok := b.sessions.TaskAt(userID, taskIdx)
		if !ok {
			b.sessions.Reset(userID)
			return sendHTML(c, "Сессия устарела. Откройте задачу снова.", mainMenu())
		}
		if err := b.client.UpdateTitle(ctx, t.ListID, t.ID, text); err != nil {
			return b.sendErr(c, err)
		}
		b.sessions.ClearPending(userID)
		updated, err := b.client.GetTask(ctx, t.ListID, t.ID)
		if err != nil {
			return sendHTML(c, "✏️ Название обновлено", mainMenu())
		}
		updated.ListName = t.ListName
		b.sessions.UpdateTaskInView(userID, *updated)
		return sendHTML(c, "✏️ <b>Название обновлено</b>\n\n"+buildTaskDetail(*updated, b.cfg.Timezone), taskDetailButtons(*updated, strconv.Itoa(taskIdx)))

	case pendingCustomDueCreate:
		due, err := ParseDueInput(text, b.cfg.Timezone)
		if err != nil {
			return sendHTML(c, "❌ "+htmlEsc(err.Error())+"\n\nФормат: <code>15.06</code>, <code>15.06.2026</code>, <code>+3d</code>", nil)
		}
		b.sessions.SetDueTime(userID, due)
		b.sessions.StartNotesCreate(userID)
		return sendHTML(c, "📝 <b>Заметка</b> (необязательно)\n\nВведите текст или нажмите «Пропустить»:", notesCreateButtons())

	case pendingCustomDueEdit:
		t, ok := b.sessions.TaskAt(userID, taskIdx)
		if !ok {
			b.sessions.Reset(userID)
			return sendHTML(c, "Сессия устарела. Откройте задачу снова.", mainMenu())
		}
		due, err := ParseDueInput(text, b.cfg.Timezone)
		if err != nil {
			return sendHTML(c, "❌ "+htmlEsc(err.Error())+"\n\nФормат: <code>15.06</code>, <code>+3d</code>", dueButtonsForTask(strconv.Itoa(taskIdx)))
		}
		if err := b.client.UpdateDue(ctx, t.ListID, t.ID, due); err != nil {
			return b.sendErr(c, err)
		}
		b.sessions.ClearPending(userID)
		updated, err := b.client.GetTask(ctx, t.ListID, t.ID)
		if err != nil {
			return sendHTML(c, "📅 Срок обновлён", mainMenu())
		}
		updated.ListName = t.ListName
		b.sessions.UpdateTaskInView(userID, *updated)
		idx := strconv.Itoa(taskIdx)
		return sendHTML(c, buildTaskDetail(*updated, b.cfg.Timezone), taskDetailButtons(*updated, idx))
	}
	return nil
}

func (b *Bot) finishCreateTask(c tele.Context, notes string) error {
	userID := c.Sender().ID
	_, title, listID, listName := b.sessions.Snapshot(userID)
	due := b.sessions.DueTime(userID)
	ctx := context.Background()

	created, err := b.client.CreateTask(ctx, listID, title, notes, due)
	if err != nil {
		b.sessions.Reset(userID)
		return b.sendErr(c, err)
	}
	b.sessions.Reset(userID)
	created.ListName = listName
	msg := "✅ <b>Создано</b>\n\n" + tasks.FormatTaskHTML(*created, b.cfg.Timezone, 1)
	return sendHTML(c, msg, afterCreateButtons())
}

func (b *Bot) sendErr(c tele.Context, err error) error {
	log.Printf("handler error: %v", err)
	msg := "❌ " + htmlEsc(err.Error())
	if c.Callback() != nil {
		_ = c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
		return editHTML(c, msg, nil)
	}
	return sendHTML(c, msg, mainMenu())
}

func htmlEsc(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
