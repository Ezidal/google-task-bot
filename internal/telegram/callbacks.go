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
		action, title, listID, listName := b.sessions.Snapshot(userID)
		if action != pendingListPick || title == "" || listID == "" {
			return c.Respond(&tele.CallbackResponse{Text: "Сессия устарела"})
		}
		var dueTime *time.Time
		if data != "none" {
			dueTime = DueFromPreset(data, b.cfg.Timezone)
		}
		created, err := b.client.CreateTask(ctx, listID, title, "", dueTime)
		if err != nil {
			b.sessions.Reset(userID)
			return b.callbackErr(c, err)
		}
		b.sessions.Reset(userID)
		created.ListName = listName
		msg := "✅ <b>Создано</b>\n\n" + tasks.FormatTaskHTML(*created, b.cfg.Timezone, 1)
		_ = replyOrEdit(c, msg, afterCreateButtons())
		return c.Respond(&tele.CallbackResponse{Text: "Готово"})

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
