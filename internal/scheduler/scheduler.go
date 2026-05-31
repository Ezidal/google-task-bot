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
}

func New(cfg *config.Config, client *tasks.Client, bot *tele.Bot, store *notify.Store) *Notifier {
	users := make([]int64, 0, len(cfg.AllowedUserIDs))
	for id := range cfg.AllowedUserIDs {
		users = append(users, id)
	}
	return &Notifier{cfg: cfg, client: client, bot: bot, store: store, users: users}
}

func (n *Notifier) Start(ctx context.Context) {
	c := cron.New(cron.WithLocation(n.cfg.Timezone))

	parts := strings.Split(n.cfg.MorningNotifyTime, ":")
	cronExpr := fmt.Sprintf("%s %s * * *", parts[1], parts[0]) // minute hour

	_, err := c.AddFunc(cronExpr, func() {
		if err := n.sendMorningSummary(ctx); err != nil {
			log.Printf("morning notify: %v", err)
		}
	})
	if err != nil {
		log.Printf("morning cron: %v", err)
	}
	c.Start()

	go func() {
		ticker := time.NewTicker(n.cfg.PollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n.checkDeadlines(ctx)
				n.checkNoDueReminders(ctx)
			}
		}
	}()
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
	n.broadcastHTML(msg)
	return nil
}

func (n *Notifier) checkDeadlines(ctx context.Context) {
	all, err := n.client.ListAllActiveTasks(ctx)
	if err != nil {
		log.Printf("deadline check: %v", err)
		return
	}
	now := time.Now().In(n.cfg.Timezone)
	for _, t := range all {
		if t.Due == nil {
			continue
		}
		due := t.Due.In(n.cfg.Timezone)
		diff := due.Sub(now)
		if diff <= time.Hour && diff > 0 {
			key := "deadline:" + t.ListID + ":" + t.ID
			if n.store.WasSent(key, 2*time.Hour) {
				continue
			}
			msg := fmt.Sprintf("⏰ <b>Через час дедлайн</b>\n\n%s", tasks.FormatTaskHTML(t, n.cfg.Timezone, 1))
			n.broadcastHTML(msg)
			n.store.Mark(key)
		}
	}
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
		if !n.store.ShouldRemindNoDue(key, n.cfg.ReminderInterval) {
			continue
		}
		msg := fmt.Sprintf("📌 <b>Без дедлайна</b>\n\n%s", tasks.FormatTaskHTML(t, n.cfg.Timezone, 1))
			n.broadcastHTML(msg)
		n.store.Mark(key)
	}
}

func (n *Notifier) broadcastHTML(text string) {
	for _, userID := range n.users {
		_, err := n.bot.Send(&tele.User{ID: userID}, text, tele.ModeHTML)
		if err != nil {
			log.Printf("send to %d: %v", userID, err)
		}
	}
}
