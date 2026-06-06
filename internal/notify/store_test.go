package notify

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStorePersistAndSnooze(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	s.Mark("k1")
	if !s.WasSent("k1", time.Hour) {
		t.Fatal("expected sent")
	}

	ref, err := s.RefTask("list1", "task1")
	if err != nil || ref < 1 {
		t.Fatalf("ref=%d err=%v", ref, err)
	}
	listID, taskID, ok := s.TaskFromRef(ref)
	if !ok || listID != "list1" || taskID != "task1" {
		t.Fatalf("got %s %s ok=%v", listID, taskID, ok)
	}

	until := time.Now().Add(time.Hour)
	s.SnoozeRef(ref, until)
	if !s.IsRefSnoozed(ref) {
		t.Fatal("expected snoozed")
	}

	loc, _ := time.LoadLocation("Europe/Moscow")
	if !s.ShouldRemindNoDue("nodue:x", time.Hour, loc, 1) {
		t.Fatal("first remind expected")
	}
	s.Mark("nodue:x")
	if s.ShouldRemindNoDue("nodue:x", time.Hour, loc, 1) {
		t.Fatal("max per day should block")
	}
}
