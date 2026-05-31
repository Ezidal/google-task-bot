package telegram

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/sergey/GoogleTaskBot/internal/tasks"
	tele "gopkg.in/telebot.v3"
)

const tasksPerPage = 8

func sendHTML(c tele.Context, text string, markup *tele.ReplyMarkup) error {
	opts := []interface{}{tele.ModeHTML}
	if markup != nil {
		opts = append(opts, markup)
	}
	return c.Send(text, opts...)
}

func editHTML(c tele.Context, text string, markup *tele.ReplyMarkup) error {
	opts := []interface{}{tele.ModeHTML}
	if markup != nil {
		opts = append(opts, markup)
	}
	return c.Edit(text, opts...)
}

func buildListMessage(title string, items []tasks.Task, loc *time.Location, page int) (string, int) {
	tasks.SortForDisplay(items, loc)
	totalPages := max(1, (len(items)+tasksPerPage-1)/tasksPerPage)
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	var body strings.Builder
	body.WriteString(tasks.FormatSummaryHeader(title, items, loc))
	body.WriteString("\n\n")

	if len(items) == 0 {
		body.WriteString("<i>Нет задач</i>")
		return body.String(), totalPages
	}

	start := page * tasksPerPage
	end := start + tasksPerPage
	if end > len(items) {
		end = len(items)
	}

	for i := start; i < end; i++ {
		body.WriteString(tasks.FormatTaskHTML(items[i], loc, i+1))
		body.WriteByte('\n')
	}

	if totalPages > 1 {
		body.WriteString(fmt.Sprintf("\n<i>Стр. %d из %d</i>", page+1, totalPages))
	}
	return body.String(), totalPages
}

func buildTaskDetail(t tasks.Task, loc *time.Location) string {
	kind := tasks.DueKindOf(t, loc, time.Now())
	badge := tasks.DueBadge(kind)
	label := tasks.DueLabel(t, loc)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s <b>%s</b>\n", badge, html.EscapeString(label)))
	if t.ListName != "" {
		b.WriteString(fmt.Sprintf("📂 %s\n\n", html.EscapeString(t.ListName)))
	}
	title := strings.TrimSpace(t.Title)
	if title == "" {
		title = "(без названия)"
	}
	b.WriteString(fmt.Sprintf("<b>%s</b>\n", html.EscapeString(title)))
	b.WriteString(fmt.Sprintf("\n📅 %s", html.EscapeString(tasks.FormatDueShort(t, loc))))
	if strings.TrimSpace(t.Notes) != "" {
		b.WriteString(fmt.Sprintf("\n\n📝 %s", html.EscapeString(t.Notes)))
	}
	if t.Status == "completed" && t.Completed != nil {
		b.WriteString(fmt.Sprintf("\n\n✅ Выполнено: %s", html.EscapeString(t.Completed.In(loc).Format("02.01.2006 15:04"))))
	}
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
