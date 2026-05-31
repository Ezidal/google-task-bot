package telegram

import (
	"testing"

	"github.com/sergey/GoogleTaskBot/internal/tasks"
)

func TestPickerListByIndex(t *testing.T) {
	s := NewSessionStore()
	uid := int64(1)
	lists := []tasks.TaskList{{ID: "a", Title: "A"}, {ID: "b", Title: "B"}}
	s.SetPickerLists(uid, lists)

	got, ok := s.PickerListAt(uid, 1)
	if !ok || got.ID != "b" {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
}

func TestClearPendingKeepsView(t *testing.T) {
	s := NewSessionStore()
	uid := int64(2)
	s.SetView(uid, &viewState{Kind: viewAll, Tasks: []tasks.Task{{ID: "t1", Title: "x"}}})
	s.StartNotesEdit(uid, 0)
	s.ClearPending(uid)

	action, _ := s.PendingTaskIndex(uid)
	if action != pendingNone {
		t.Fatalf("action=%v", action)
	}
	v := s.View(uid)
	if v == nil || len(v.Tasks) != 1 {
		t.Fatalf("view lost: %+v", v)
	}
}

func TestUpdateTaskInView(t *testing.T) {
	s := NewSessionStore()
	uid := int64(3)
	s.SetView(uid, &viewState{
		Tasks: []tasks.Task{{ID: "t1", ListID: "l1", Title: "old", ListName: "L"}},
	})
	s.SetTaskIndex(uid, 0)
	s.UpdateTaskInView(uid, tasks.Task{ID: "t1", ListID: "l1", Title: "new", Notes: "n"})

	tk, ok := s.CurrentTask(uid)
	if !ok || tk.Title != "new" || tk.Notes != "n" || tk.ListName != "L" {
		t.Fatalf("got %+v", tk)
	}
}
