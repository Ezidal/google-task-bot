package telegram

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/sergey/GoogleTaskBot/internal/tasks"
	tele "gopkg.in/telebot.v3"
)

// telebot заполняет Callback.Unique только при отдельном Handle на каждый unique.
// При общем OnCallback в Data приходит "\f<unique>|<payload>".
var callbackDataRx = regexp.MustCompile(`^\f([-\w]+)(\|(.+))?$`)

func decodeCallback(cb *tele.Callback) (unique, payload string) {
	if cb == nil {
		return "", ""
	}
	if cb.Unique != "" {
		return cb.Unique, cb.Data
	}
	raw := cb.Data
	if m := callbackDataRx.FindStringSubmatch(raw); m != nil {
		return m[1], m[3]
	}
	return "", raw
}

func (b *Bot) taskFromCallback(c tele.Context, userID int64, payload string) (tasks.Task, bool) {
	if idx, err := strconv.Atoi(payload); err == nil {
		if t, ok := b.sessions.TaskAt(userID, idx); ok {
			b.sessions.SetTaskIndex(userID, idx)
			return t, true
		}
	}
	return b.sessions.CurrentTask(userID)
}

// splitTaskPayload разбирает "3:today" (tdue) или просто "3".
func splitTaskPayload(data string) (taskIdx, preset string) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) == 2 && parts[1] != "" {
		return parts[0], parts[1]
	}
	return data, ""
}
