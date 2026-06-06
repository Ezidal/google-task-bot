package tasks

import (
	"testing"
	"time"
)

func TestFilterTomorrowAndSearch(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	now := time.Now().In(loc)
	tomorrow := now.AddDate(0, 0, 1)
	tomNoon := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 12, 0, 0, 0, time.UTC)

	items := []Task{
		{ID: "1", Title: "Buy milk", Due: &tomNoon},
		{ID: "2", Title: "Other", Notes: "milk powder"},
		{ID: "3", Title: "Later"},
	}

	tom := FilterTomorrow(items, loc)
	if len(tom) != 1 || tom[0].ID != "1" {
		t.Fatalf("tomorrow: %+v", tom)
	}

	found := FilterSearch(items, "milk")
	if len(found) != 2 {
		t.Fatalf("search: %+v", found)
	}
}
