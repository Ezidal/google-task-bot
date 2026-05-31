package tasks

import (
	"fmt"
	"html"
	"sort"
	"strings"
	"time"
)

type DueKind int

const (
	DueNone DueKind = iota
	DueOverdue
	DueToday
	DueTomorrow
	DueUpcoming
)

func DueKindOf(t Task, loc *time.Location, now time.Time) DueKind {
	if t.Due == nil || t.Status == "completed" {
		return DueNone
	}
	today := dateOnly(now.In(loc), loc)
	due := dateOnly(t.Due.In(loc), loc)
	if due.Before(today) {
		return DueOverdue
	}
	if due.Equal(today) {
		return DueToday
	}
	tomorrow := today.AddDate(0, 0, 1)
	if due.Equal(tomorrow) {
		return DueTomorrow
	}
	return DueUpcoming
}

func IsOverdue(t Task, loc *time.Location) bool {
	return DueKindOf(t, loc, time.Now()) == DueOverdue
}

func IsDueToday(t Task, loc *time.Location) bool {
	return DueKindOf(t, loc, time.Now()) == DueToday
}

func dateOnly(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func dueSortPriority(k DueKind) int {
	switch k {
	case DueOverdue:
		return 0
	case DueToday:
		return 1
	case DueTomorrow:
		return 2
	case DueUpcoming:
		return 3
	default:
		return 4
	}
}

func SortForDisplay(items []Task, loc *time.Location) {
	now := time.Now()
	sort.SliceStable(items, func(i, j int) bool {
		pi := dueSortPriority(DueKindOf(items[i], loc, now))
		pj := dueSortPriority(DueKindOf(items[j], loc, now))
		if pi != pj {
			return pi < pj
		}
		di, dj := items[i].Due, items[j].Due
		if di != nil && dj != nil {
			return di.Before(*dj)
		}
		if di != nil {
			return true
		}
		if dj != nil {
			return false
		}
		if items[i].ListName != items[j].ListName {
			return items[i].ListName < items[j].ListName
		}
		return items[i].Title < items[j].Title
	})
}

func DueBadge(kind DueKind) string {
	switch kind {
	case DueOverdue:
		return "🔴"
	case DueToday:
		return "🟡"
	case DueTomorrow:
		return "🔵"
	case DueUpcoming:
		return "🟢"
	default:
		return "⚪"
	}
}

func DueLabel(t Task, loc *time.Location) string {
	kind := DueKindOf(t, loc, time.Now())
	switch kind {
	case DueOverdue:
		return "просрочено"
	case DueToday:
		return "сегодня"
	case DueTomorrow:
		return "завтра"
	case DueUpcoming:
		if t.Due != nil {
			return t.Due.In(loc).Format("02.01.2006")
		}
		return ""
	default:
		return "без срока"
	}
}

func FormatDueShort(t Task, loc *time.Location) string {
	if t.Due == nil {
		return "без срока"
	}
	kind := DueKindOf(t, loc, time.Now())
	switch kind {
	case DueOverdue:
		days := int(dateOnly(time.Now(), loc).Sub(dateOnly(t.Due.In(loc), loc)).Hours() / 24)
		if days < 1 {
			days = 1
		}
		return fmt.Sprintf("просрочено %d дн.", days)
	case DueToday:
		return "сегодня"
	case DueTomorrow:
		return "завтра"
	default:
		return t.Due.In(loc).Format("02.01.2006")
	}
}

func FormatTaskHTML(t Task, loc *time.Location, index int) string {
	badge := DueBadge(DueKindOf(t, loc, time.Now()))
	title := html.EscapeString(strings.TrimSpace(t.Title))
	if title == "" {
		title = "(без названия)"
	}
	line := fmt.Sprintf("%s <b>%d.</b> %s", badge, index, title)
	if t.ListName != "" {
		line += fmt.Sprintf("\n    <i>%s</i>", html.EscapeString(t.ListName))
	}
	line += fmt.Sprintf("\n    📅 %s", html.EscapeString(FormatDueShort(t, loc)))
	if strings.TrimSpace(t.Notes) != "" {
		note := t.Notes
		if len([]rune(note)) > 80 {
			note = string([]rune(note)[:80]) + "…"
		}
		line += fmt.Sprintf("\n    📝 %s", html.EscapeString(note))
	}
	return line
}

func FormatTaskLine(t Task, loc *time.Location) string {
	badge := DueBadge(DueKindOf(t, loc, time.Now()))
	var b strings.Builder
	b.WriteString(badge)
	b.WriteString(" ")
	if t.ListName != "" {
		b.WriteString("[")
		b.WriteString(t.ListName)
		b.WriteString("] ")
	}
	b.WriteString(t.Title)
	b.WriteString(" — ")
	b.WriteString(FormatDueShort(t, loc))
	return b.String()
}

func FormatSummaryHeader(title string, items []Task, loc *time.Location) string {
	var overdue, today, nodue int
	for _, t := range items {
		switch DueKindOf(t, loc, time.Now()) {
		case DueOverdue:
			overdue++
		case DueToday:
			today++
		case DueNone:
			if t.Status != "completed" {
				nodue++
			}
		}
	}
	return fmt.Sprintf(
		"<b>%s</b>\n\n"+
			"🔴 просрочено: <b>%d</b>  ·  🟡 сегодня: <b>%d</b>  ·  ⚪ без срока: <b>%d</b>  ·  всего: <b>%d</b>",
		html.EscapeString(title), overdue, today, nodue, len(items),
	)
}

func FilterOverdue(items []Task, loc *time.Location) []Task {
	var out []Task
	for _, t := range items {
		if IsOverdue(t, loc) {
			out = append(out, t)
		}
	}
	return out
}

func FilterToday(items []Task, loc *time.Location) []Task {
	var out []Task
	for _, t := range items {
		if IsDueToday(t, loc) {
			out = append(out, t)
		}
	}
	return out
}
