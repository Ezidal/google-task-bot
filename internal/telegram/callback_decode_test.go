package telegram

import (
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestDecodeCallbackTelebotFormat(t *testing.T) {
	u, p := decodeCallback(&tele.Callback{Data: "\fdone|3"})
	if u != "done" || p != "3" {
		t.Fatalf("got unique=%q payload=%q", u, p)
	}
}

func TestDecodeCallbackTduePayload(t *testing.T) {
	u, p := decodeCallback(&tele.Callback{Data: "\ftdue|2:tomorrow"})
	if u != "tdue" || p != "2:tomorrow" {
		t.Fatalf("got unique=%q payload=%q", u, p)
	}
	idx, preset := splitTaskPayload(p)
	if idx != "2" || preset != "tomorrow" {
		t.Fatalf("split got idx=%q preset=%q", idx, preset)
	}
}

func TestDecodeCallbackPreParsed(t *testing.T) {
	u, p := decodeCallback(&tele.Callback{Unique: "open", Data: "5"})
	if u != "open" || p != "5" {
		t.Fatalf("got unique=%q payload=%q", u, p)
	}
}
