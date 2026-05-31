package telegram

import (
	"log"
	"net/http"
	"time"

	"github.com/sergey/GoogleTaskBot/internal/config"
	"github.com/sergey/GoogleTaskBot/internal/tasks"
	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	cfg      *config.Config
	client   *tasks.Client
	bot      *tele.Bot
	sessions *SessionStore
}

func New(cfg *config.Config, client *tasks.Client, httpClient *http.Client) (*Bot, error) {
	pref := tele.Settings{
		Token:  cfg.TelegramToken,
		Client: httpClient,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}
	return &Bot{cfg: cfg, client: client, bot: b, sessions: NewSessionStore()}, nil
}

func (b *Bot) TeleBot() *tele.Bot {
	return b.bot
}

func (b *Bot) Run() {
	b.bot.Use(b.authMiddleware)

	b.bot.Handle("/start", b.onStart)
	b.bot.Handle("/help", b.onHelp)
	b.bot.Handle("/tasks", b.onAllTasks)
	b.bot.Handle("/overdue", b.onOverdue)
	b.bot.Handle("/today", b.onTodayTasks)
	b.bot.Handle("/lists", b.onListsMenu)
	b.bot.Handle("/completed", b.onCompleted)
	b.bot.Handle("/add", b.onAddStart)

	b.bot.Handle(&tele.Btn{Text: "📋 Все задачи"}, b.onAllTasks)
	b.bot.Handle(&tele.Btn{Text: "🔴 Просроченные"}, b.onOverdue)
	b.bot.Handle(&tele.Btn{Text: "📅 На сегодня"}, b.onTodayTasks)
	b.bot.Handle(&tele.Btn{Text: "📂 Списки"}, b.onListsMenu)
	b.bot.Handle(&tele.Btn{Text: "➕ Добавить"}, b.onAddStart)
	b.bot.Handle(&tele.Btn{Text: "✅ Выполненные"}, b.onCompleted)

	b.bot.Handle(tele.OnCallback, b.onCallback)
	b.bot.Handle(tele.OnText, b.onText)

	log.Println("telegram bot started")
	b.bot.Start()
}

func (b *Bot) authMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if c.Sender() == nil {
			return nil
		}
		if _, ok := b.cfg.AllowedUserIDs[c.Sender().ID]; !ok {
			return c.Send("Доступ запрещён.")
		}
		return next(c)
	}
}
