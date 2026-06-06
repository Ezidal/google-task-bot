package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sergey/GoogleTaskBot/internal/tasks"
	tele "gopkg.in/telebot.v3"
)

func (b *Bot) onCallback(c tele.Context) error {
	unique, data := decodeCallback(c.Callback())
	userID := c.Sender().ID
	ctx := context.Background()

	switch unique {
	case "cancel":
		if data == "menu" {
			b.sessions.Reset(userID)
			_ = c.Edit(" ")
			return c.Send("🏠 Меню", mainMenu())
		}
		b.sessions.Reset(userID)
		_ = c.Respond(&tele.CallbackResponse{Text: "Отменено"})
		return replyOrEdit(c, "❌ Добавление отменено", nil)

	case "picklist":
		idx, err := strconv.Atoi(data)
		if err != nil {
			return c.Respond()
		}
		list, ok := b.sessions.PickerListAt(userID, idx)
		if !ok {
			return c.Respond(&tele.CallbackResponse{Text: "Обновите список"})
		}
		action, title, _, _ := b.sessions.Snapshot(userID)
		if action == pendingListPick && title != "" {
			b.sessions.SetList(userID, list.ID, list.Title)
			return replyOrEdit(c, fmt.Sprintf("📋 <b>%s</b>\n\nВыберите срок:", htmlEsc(title)), dueButtons())
		}
		return b.loadAndShow(c, viewList, list.ID, list.Title, 0)

	case "due":
		action, title, listID, _ := b.sessions.Snapshot(userID)
		if action != pendingListPick || title == "" || listID == "" {
			return c.Respond(&tele.CallbackResponse{Text: "Сессия устарела"})
		}
		var dueTime *time.Time
		if data != "none" {
			dueTime = DueFromPreset(data, b.cfg.Timezone)
		}
		b.sessions.SetDueTime(userID, dueTime)
		b.sessions.StartNotesCreate(userID)
		if err := replyOrEdit(c, "📝 <b>Заметка</b> (необязательно)\n\nВведите текст или нажмите «Пропустить»:", notesCreateButtons()); err != nil {
			return err
		}
		return c.Respond()

	case "skipnotes":
		if err := b.finishCreateTask(c, ""); err != nil {
			return err
		}
		return c.Respond(&tele.CallbackResponse{Text: "Готово"})

	case "customdue":
		b.sessions.StartCustomDueCreate(userID)
		_ = c.Respond()
		return c.Send("📆 Введите дату:\n<code>15.06</code>, <code>15.06.2026</code> или <code>+3d</code>", tele.ModeHTML)

	case "tcustom":
		idx, err := strconv.Atoi(data)
		if err != nil {
			return c.Respond()
		}
		b.sessions.StartCustomDueEdit(userID, idx)
		_ = c.Respond()
		return c.Send("📆 Введите дату:\n<code>15.06</code> или <code>+3d</code>", tele.ModeHTML)

	case "open":
		idx, err := strconv.Atoi(data)
		if err != nil {
			return c.Respond()
		}
		if err := b.openTask(c, idx); err != nil {
			return err
		}
		return c.Respond()

	case "page":
		parts := strings.SplitN(data, ":", 2)
		if len(parts) != 2 {
			return c.Respond()
		}
		page, _ := strconv.Atoi(parts[1])
		v := b.sessions.View(userID)
		if v == nil {
			return c.Respond(&tele.CallbackResponse{Text: "Обновите"})
		}
		return b.loadAndShow(c, viewKind(parts[0]), v.ListID, v.ListName, page)

	case "refresh":
		v := b.sessions.View(userID)
		if v == nil {
			return b.loadAndShow(c, viewKind(data), "", "", 0)
		}
		return b.loadAndShow(c, v.Kind, v.ListID, v.ListName, v.Page)

	case "clear":
		v := b.sessions.View(userID)
		if v == nil || v.ListID == "" {
			return c.Respond(&tele.CallbackResponse{Text: "Нет списка"})
		}
		if err := b.client.ClearCompleted(ctx, v.ListID); err != nil {
			return b.callbackErr(c, err)
		}
		_ = c.Respond(&tele.CallbackResponse{Text: "Выполненные очищены"})
		return b.loadAndShow(c, viewList, v.ListID, v.ListName, v.Page)

	case "back":
		if err := b.reloadCurrentView(c); err != nil {
			return err
		}
		return c.Respond()

	case "done":
		return b.taskAction(c, ctx, data, func(t tasks.Task) error {
			return b.client.CompleteTask(ctx, t.ListID, t.ID)
		}, "✅ Выполнено")

	case "reopen":
		return b.taskAction(c, ctx, data, func(t tasks.Task) error {
			return b.client.ReopenTask(ctx, t.ListID, t.ID)
		}, "↩️ В работе")

	case "delete":
		t, ok := b.taskFromCallback(c, userID, data)
		if !ok {
			return c.Respond(&tele.CallbackResponse{Text: "Обновите список"})
		}
		if err := b.client.DeleteTask(ctx, t.ListID, t.ID); err != nil {
			return b.callbackErr(c, err)
		}
		_ = c.Respond(&tele.CallbackResponse{Text: "Удалено"})
		return b.reloadCurrentView(c)

	case "dueedit":
		if _, ok := b.taskFromCallback(c, userID, data); !ok {
			return c.Respond(&tele.CallbackResponse{Text: "Обновите список"})
		}
		return replyOrEdit(c, "📅 <b>Выберите срок</b>", dueButtonsForTask(data))

	case "tdue":
		taskIdx, preset := splitTaskPayload(data)
		t, ok := b.taskFromCallback(c, userID, taskIdx)
		if !ok {
			return c.Respond(&tele.CallbackResponse{Text: "Обновите список"})
		}
		var dueTime *time.Time
		if preset != "none" && preset != "" {
			dueTime = DueFromPreset(preset, b.cfg.Timezone)
		}
		if err := b.client.UpdateDue(ctx, t.ListID, t.ID, dueTime); err != nil {
			return b.callbackErr(c, err)
		}
		updated, err := b.client.GetTask(ctx, t.ListID, t.ID)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "Срок обновлён"})
		}
		updated.ListName = t.ListName
		b.sessions.UpdateTaskInView(userID, *updated)
		b.sessions.ClearPending(userID)
		if taskIdx == "" {
			taskIdx = strconv.Itoa(findTaskIndex(b.sessions.View(userID), t.ID))
		}
		_ = replyOrEdit(c, buildTaskDetail(*updated, b.cfg.Timezone), taskDetailButtons(*updated, taskIdx))
		return c.Respond(&tele.CallbackResponse{Text: "Готово"})

	case "rename":
		idx, err := strconv.Atoi(data)
		if err != nil {
			return c.Respond()
		}
		t, ok := b.sessions.TaskAt(userID, idx)
		if !ok {
			return c.Respond(&tele.CallbackResponse{Text: "Обновите список"})
		}
		b.sessions.StartRename(userID, idx)
		_ = c.Respond()
		return c.Send(fmt.Sprintf("✏️ Новое название для:\n<b>%s</b>", htmlEsc(t.Title)), tele.ModeHTML)

	case "notes":
		t, ok := b.taskFromCallback(c, userID, data)
		if !ok {
			return c.Respond(&tele.CallbackResponse{Text: "Обновите список"})
		}
		idx, _ := strconv.Atoi(data)
		if idx < 0 {
			idx = findTaskIndex(b.sessions.View(userID), t.ID)
		}
		b.sessions.StartNotesEdit(userID, idx)
		_ = c.Respond()
		return c.Send(
			fmt.Sprintf("📝 Введите заметку для:\n<b>%s</b>", htmlEsc(t.Title)),
			tele.ModeHTML,
		)

	case "clrnotes":
		t, ok := b.taskFromCallback(c, userID, data)
		if !ok {
			return c.Respond(&tele.CallbackResponse{Text: "Обновите список"})
		}
		if err := b.client.UpdateNotes(ctx, t.ListID, t.ID, ""); err != nil {
			return b.callbackErr(c, err)
		}
		updated, err := b.client.GetTask(ctx, t.ListID, t.ID)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "Заметки удалены"})
		}
		updated.ListName = t.ListName
		b.sessions.UpdateTaskInView(userID, *updated)
		taskIdx := data
		_ = replyOrEdit(c, buildTaskDetail(*updated, b.cfg.Timezone), taskDetailButtons(*updated, taskIdx))
		return c.Respond(&tele.CallbackResponse{Text: "Заметки удалены"})

	case "nact":
		return b.onNotifyAction(c, ctx, data)
	}
	return c.Respond()
}

func (b *Bot) onNotifyAction(c tele.Context, ctx context.Context, data string) error {
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 2 {
		return c.Respond()
	}
	refID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return c.Respond()
	}
	listID, taskID, ok := b.notify.TaskFromRef(refID)
	if !ok {
		return c.Respond(&tele.CallbackResponse{Text: "Задача не найдена"})
	}

	switch parts[0] {
	case "d":
		if err := b.client.CompleteTask(ctx, listID, taskID); err != nil {
			return b.callbackErr(c, err)
		}
		_ = c.Respond(&tele.CallbackResponse{Text: "✅ Выполнено"})
		return editHTML(c, "✅ Задача выполнена", nil)

	case "o":
		_ = c.Respond()
		return b.openTaskByRef(c, listID, taskID)

	case "s":
		if len(parts) < 3 {
			return c.Respond()
		}
		loc := b.cfg.Timezone
		now := time.Now().In(loc)
		var until time.Time
		switch parts[2] {
		case "1h":
			until = now.Add(time.Hour)
		case "tom":
			tom := now.AddDate(0, 0, 1)
			until = time.Date(tom.Year(), tom.Month(), tom.Day(), 9, 0, 0, 0, loc)
		default:
			return c.Respond()
		}
		b.notify.SnoozeRef(refID, until)
		_ = c.Respond(&tele.CallbackResponse{Text: "Отложено"})
		return nil
	}
	return c.Respond()
}

func (b *Bot) taskAction(c tele.Context, ctx context.Context, payload string, fn func(tasks.Task) error, okText string) error {
	t, ok := b.taskFromCallback(c, c.Sender().ID, payload)
	if !ok {
		return c.Respond(&tele.CallbackResponse{Text: "Обновите список"})
	}
	if err := fn(t); err != nil {
		return b.callbackErr(c, err)
	}
	_ = c.Respond(&tele.CallbackResponse{Text: okText})
	return b.reloadCurrentView(c)
}

func (b *Bot) callbackErr(c tele.Context, err error) error {
	_ = c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	return b.sendErr(c, err)
}
