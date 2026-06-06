package telegram

import (
	"fmt"

	"github.com/sergey/GoogleTaskBot/internal/tasks"
	tele "gopkg.in/telebot.v3"
)

func mainMenu() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(menu.Text("📋 Все задачи"), menu.Text("🔴 Просроченные")),
		menu.Row(menu.Text("📅 На сегодня"), menu.Text("🔵 На завтра")),
		menu.Row(menu.Text("📂 Списки"), menu.Text("➕ Добавить")),
		menu.Row(menu.Text("✅ Выполненные")),
	)
	return menu
}

func dueButtons() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("Сегодня", "due", "today"),
			markup.Data("Завтра", "due", "tomorrow"),
		),
		markup.Row(
			markup.Data("Через неделю", "due", "week"),
			markup.Data("Без срока", "due", "none"),
		),
		markup.Row(markup.Data("📆 Другая дата", "customdue", "add")),
		markup.Row(markup.Data("« Отмена", "cancel", "add")),
	)
	return markup
}

func dueButtonsForTask(taskIdx string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	prefix := taskIdx + ":"
	markup.Inline(
		markup.Row(
			markup.Data("Сегодня", "tdue", prefix+"today"),
			markup.Data("Завтра", "tdue", prefix+"tomorrow"),
		),
		markup.Row(
			markup.Data("Неделя", "tdue", prefix+"week"),
			markup.Data("Убрать срок", "tdue", prefix+"none"),
		),
		markup.Row(markup.Data("📆 Другая дата", "tcustom", taskIdx)),
		markup.Row(markup.Data("« Назад", "back", "task")),
	)
	return markup
}

func notesCreateButtons() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("Пропустить", "skipnotes", "create")),
		markup.Row(markup.Data("« Отмена", "cancel", "add")),
	)
	return markup
}

func listPickerButtons(lists []tasks.TaskList) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	for i, l := range lists {
		rows = append(rows, markup.Row(markup.Data("📂 "+truncate(l.Title, 28), "picklist", fmt.Sprintf("%d", i))))
	}
	rows = append(rows, markup.Row(markup.Data("« Меню", "cancel", "menu")))
	markup.Inline(rows...)
	return markup
}

func taskListButtons(items []tasks.Task, page, totalPages int, viewKind viewKind) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row

	start := page * tasksPerPage
	end := start + tasksPerPage
	if end > len(items) {
		end = len(items)
	}

	for i := start; i < end; i++ {
		title := truncate(items[i].Title, 24)
		if title == "" {
			title = "…"
		}
		rows = append(rows, markup.Row(
			markup.Data("☑ "+title, "open", fmt.Sprintf("%d", i)),
		))
	}

	var nav []tele.Btn
	if page > 0 {
		nav = append(nav, markup.Data("◀️", "page", fmt.Sprintf("%s:%d", viewKind, page-1)))
	}
	if page < totalPages-1 {
		nav = append(nav, markup.Data("▶️", "page", fmt.Sprintf("%s:%d", viewKind, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, markup.Row(nav...))
	}

	rows = append(rows, markup.Row(markup.Data("🔄 Обновить", "refresh", string(viewKind))))
	if viewKind == viewList {
		rows = append(rows, markup.Row(markup.Data("🗑 Очистить выполненные", "clear", "list")))
	}
	rows = append(rows, markup.Row(markup.Data("« Меню", "cancel", "menu")))
	markup.Inline(rows...)
	return markup
}

func taskDetailButtons(t tasks.Task, taskIdx string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	if t.Status == "completed" {
		rows := []tele.Row{
			markup.Row(markup.Data("↩️ Вернуть", "reopen", taskIdx)),
			markup.Row(markup.Data("🗑 Удалить", "delete", taskIdx)),
			markup.Row(markup.Data("« Назад", "back", "list")),
		}
		if t.WebLink != "" {
			rows = append([]tele.Row{markup.Row(markup.URL("🔗 Google Tasks", t.WebLink))}, rows...)
		}
		markup.Inline(rows...)
		return markup
	}
	rows := []tele.Row{
		markup.Row(
			markup.Data("✅ Выполнить", "done", taskIdx),
			markup.Data("📅 Срок", "dueedit", taskIdx),
		),
		markup.Row(
			markup.Data("✏️ Название", "rename", taskIdx),
			markup.Data("📝 Заметки", "notes", taskIdx),
		),
		markup.Row(
			markup.Data("🗑 Удалить", "delete", taskIdx),
			markup.Data("🧹 Без заметок", "clrnotes", taskIdx),
		),
		markup.Row(markup.Data("« Назад", "back", "list")),
	}
	if t.WebLink != "" {
		rows = append([]tele.Row{markup.Row(markup.URL("🔗 Google Tasks", t.WebLink))}, rows...)
	}
	markup.Inline(rows...)
	return markup
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
