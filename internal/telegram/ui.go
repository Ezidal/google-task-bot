package telegram

import tele "gopkg.in/telebot.v3"

// replyOrEdit обновляет сообщение с inline-кнопками или шлёт новое при ошибке Edit.
func replyOrEdit(c tele.Context, text string, markup *tele.ReplyMarkup) error {
	if c.Callback() != nil {
		if err := editHTML(c, text, markup); err != nil {
			return sendHTML(c, text, markup)
		}
		return nil
	}
	return sendHTML(c, text, markup)
}

func afterCreateButtons() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(m.Data("📋 Все задачи", "refresh", string(viewAll))),
		m.Row(m.Data("« Меню", "cancel", "menu")),
	)
	return m
}
