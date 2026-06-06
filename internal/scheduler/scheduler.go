package scheduler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sergey/GoogleTaskBot/internal/config"
	"github.com/sergey/GoogleTaskBot/internal/notify"
	"github.com/sergey/GoogleTaskBot/internal/tasks"
	tele "gopkg.in/telebot.v3"
)

type Notifier struct {
	cfg    *config.Config
	client *tasks.Client
	bot    *tele.Bot
	store  *notify.Store
	users  []int64
	cron   *cron.Cron
}

func New(cfg *config.Config, client *tasks.Client, bot *tele.Bot, store *notify.Store) *Notifier {
	users := make([]int64, 0, len(cfg.AllowedUserIDs))
	for id := range cfg.AllowedUserIDs {
		users = append(users, id)
	}
	return &Notifier{cfg: cfg, client: client, bot: bot, store: store, users: users}
}

func (n *Notifier) Start(ctx context.Context) {
	n.cron = cron.New(cron.WithLocation(n.cfg.Timezone))

	parts := strings.Split(n.cfg.MorningNotifyTime, ":")
	cronExpr := fmt.Sprintf("%s %s * * *", parts[1], parts[0])
	if _, err := n.cron.AddFunc(cronExpr, func() {
		if err := n.sendMorningSummary(ctx); err != nil {
			log.Printf("morning notify: %v", err)
		}
	}); err != nil {
		log.Printf("morning cron: %v", err)
	}

	n.cron.Start()

	go func() {
		ticker := time.NewTicker(n.cfg.PollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n.checkDueDayReminders(ctx)
				n.checkNoDueReminders(ctx)
			}
		}
	}()
}

func (n *Notifier) Stop() {
	if n.cron != nil {
		ctx := n.cron.Stop()
		<-ctx.Done()
	}
}

func (n *Notifier) sendMorningSummary(ctx context.Context) error {
	all, err := n.client.ListAllActiveTasks(ctx)
	if err != nil {
		return err
	}
	var today, overdue []tasks.Task
	for _, t := range all {
		if tasks.IsDueToday(t, n.cfg.Timezone) {
			today = append(today, t)
		}
		if tasks.IsOverdue(t, n.cfg.Timezone) {
			overdue = append(overdue, t)
		}
	}
	var msg string
	if len(today) == 0 && len(overdue) == 0 {
		msg = "☀️ <b>Доброе утро!</b>\n\nНа сегодня дедлайнов нет. Просроченных нет."
	} else {
		var b strings.Builder
		b.WriteString("☀️ <b>Доброе утро!</b>\n\n")
		if len(overdue) > 0 {
			b.WriteString(fmt.Sprintf("🔴 <b>Просрочено: %d</b>\n", len(overdue)))
			tasks.SortForDisplay(overdue, n.cfg.Timezone)
			for i, t := range overdue {
				if i >= 5 {
					b.WriteString(fmt.Sprintf("<i>... ещё %d</i>\n", len(overdue)-5))
					break
				}
				b.WriteString(tasks.FormatTaskHTML(t, n.cfg.Timezone, i+1))
				b.WriteByte('\n')
			}
			b.WriteByte('\n')
		}
		if len(today) > 0 {
			b.WriteString(fmt.Sprintf("🟡 <b>На сегодня: %d</b>\n", len(today)))
			tasks.SortForDisplay(today, n.cfg.Timezone)
			for i, t := range today {
				b.WriteString(tasks.FormatTaskHTML(t, n.cfg.Timezone, i+1))
				b.WriteByte('\n')
			}
		}
		msg = b.String()
	}
	n.broadcastHTML(msg, nil)
	return nil
}

func (n *Notifier) checkDueDayReminders(ctx context.Context) {
	if !n.pastDeadlineNotifyTime() {
		return
	}
	all, err := n.client.ListAllActiveTasks(ctx)
	if err != nil {
		log.Printf("due-day check: %v", err)
		return
	}
	todayKey := time.Now().In(n.cfg.Timezone).Format("2006-01-02")
	for _, t := range all {
		if t.Due == nil || !tasks.IsDueToday(t, n.cfg.Timezone) {
			continue
		}
		key := fmt.Sprintf("dueday:%s:%s:%s", t.ListID, t.ID, todayKey)
		ref, _ := n.store.RefTask(t.ListID, t.ID)
		if n.store.IsRefSnoozed(ref) || n.store.WasSent(key, 36*time.Hour) {
			continue
		}
		msg := fmt.Sprintf("📅 <b>Дедлайн сегодня</b>\n\n%s", tasks.FormatTaskHTML(t, n.cfg.Timezone, 1))
		n.broadcastHTML(msg, n.notifyMarkup(t))
		n.store.Mark(key)
	}
}

func (n *Notifier) pastDeadlineNotifyTime() bool {
	now := time.Now().In(n.cfg.Timezone)
	parts := strings.Split(n.cfg.DeadlineNotifyTime, ":")
	if len(parts) != 2 {
		return true
	}
	hour, min := 0, 0
	fmt.Sscanf(parts[0], "%d", &hour)
	fmt.Sscanf(parts[1], "%d", &min)
	threshold := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, n.cfg.Timezone)
	return !now.Before(threshold)
}

func (n *Notifier) checkNoDueReminders(ctx context.Context) {
	all, err := n.client.ListAllActiveTasks(ctx)
	if err != nil {
		log.Printf("no-due reminder: %v", err)
		return
	}
	for _, t := range all {
		if t.Due != nil {
			continue
		}
		key := "nodue:" + t.ListID + ":" + t.ID
		ref, _ := n.store.RefTask(t.ListID, t.ID)
		if n.store.IsRefSnoozed(ref) || !n.store.ShouldRemindNoDue(key, n.cfg.ReminderInterval, n.cfg.Timezone, n.cfg.NoDueMaxPerDay) {
			continue
		}
		msg := fmt.Sprintf("📌 <b>Без дедлайна</b>\n\n%s", tasks.FormatTaskHTML(t, n.cfg.Timezone, 1))
		n.broadcastHTML(msg, n.notifyMarkup(t))
		n.store.Mark(key)
	}
}

func (n *Notifier) notifyMarkup(t tasks.Task) *tele.ReplyMarkup {
	ref, err := n.store.RefTask(t.ListID, t.ID)
	if err != nil {
		return nil
	}
	m := &tele.ReplyMarkup{}
	rows := []tele.Row{
		m.Row(
			m.Data("✅ Готово", "nact", fmt.Sprintf("d:%d", ref)),
			m.Data("📋 Открыть", "nact", fmt.Sprintf("o:%d", ref)),
		),
		m.Row(
			m.Data("⏰ +1ч", "nact", fmt.Sprintf("s:1h:%d", ref)),
			m.Data("🌅 Завтра", "nact", fmt.Sprintf("s:tom:%d", ref)),
		),
	}
	if t.WebLink != "" {
		rows = append(rows, m.Row(m.URL("🔗 Google Tasks", t.WebLink)))
	}
	m.Inline(rows...)
	return m
}

func (n *Notifier) broadcastHTML(text string, markup *tele.ReplyMarkup) {
	opts := []interface{}{tele.ModeHTML}
	if markup != nil {
		opts = append(opts, markup)
	}
	for _, userID := range n.users {
		_, err := n.bot.Send(&tele.User{ID: userID}, text, opts...)
		if err != nil {
			log.Printf("send to %d: %v", userID, err)
		}
	}
}
