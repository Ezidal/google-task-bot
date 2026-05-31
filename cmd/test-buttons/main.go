// Интеграционная проверка всех операций, которые вызывают inline-кнопки.
//
//	go run ./cmd/test-buttons
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sergey/GoogleTaskBot/internal/config"
	"github.com/sergey/GoogleTaskBot/internal/httpclient"
	"github.com/sergey/GoogleTaskBot/internal/tasks"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	httpClient, err := httpclient.New(cfg.HTTPProxy)
	if err != nil {
		log.Fatal(err)
	}
	client, err := tasks.NewClient(ctx, cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRefreshToken, httpClient)
	if err != nil {
		log.Fatal(err)
	}

	lists, err := client.ListTaskLists(ctx)
	if err != nil {
		log.Fatalf("ListTaskLists: %v", err)
	}
	fmt.Printf("✓ picklist: %d списков\n", len(lists))
	if len(lists) == 0 {
		log.Fatal("нет списков")
	}
	list := lists[0]
	loc := cfg.Timezone

	created, err := client.CreateTask(ctx, list.ID, "[bot-test] inline check", "", nil)
	if err != nil {
		log.Fatalf("CreateTask: %v", err)
	}
	fmt.Printf("✓ due/create: %s\n", created.ID)

	got, err := client.GetTask(ctx, list.ID, created.ID)
	if err != nil {
		log.Fatalf("GetTask: %v", err)
	}
	fmt.Printf("✓ open: %q\n", got.Title)

	if err := client.UpdateNotes(ctx, list.ID, created.ID, "test note"); err != nil {
		log.Fatalf("UpdateNotes: %v", err)
	}
	fmt.Println("✓ notes")

	tomorrow := time.Now().In(loc).AddDate(0, 0, 1)
	due := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 12, 0, 0, 0, loc).UTC()
	if err := client.UpdateDue(ctx, list.ID, created.ID, &due); err != nil {
		log.Fatalf("UpdateDue: %v", err)
	}
	fmt.Println("✓ tdue")

	if err := client.CompleteTask(ctx, list.ID, created.ID); err != nil {
		log.Fatalf("CompleteTask: %v", err)
	}
	fmt.Println("✓ done")

	if err := client.ReopenTask(ctx, list.ID, created.ID); err != nil {
		log.Fatalf("ReopenTask: %v", err)
	}
	fmt.Println("✓ reopen")

	if err := client.DeleteTask(ctx, list.ID, created.ID); err != nil {
		log.Fatalf("DeleteTask: %v", err)
	}
	fmt.Println("✓ delete")

	all, err := client.ListAllActiveTasks(ctx)
	if err != nil {
		log.Fatalf("ListAllActiveTasks: %v", err)
	}
	fmt.Printf("✓ refresh/all: %d\n", len(all))
	fmt.Printf("✓ overdue filter: %d\n", len(tasks.FilterOverdue(all, loc)))
	fmt.Printf("✓ today filter: %d\n", len(tasks.FilterToday(all, loc)))

	raw, err := client.ListTasks(ctx, list.ID, false)
	if err != nil {
		log.Fatalf("ListTasks: %v", err)
	}
	fmt.Printf("✓ list view: %d активных в %q\n", len(raw), list.Title)

	if err := client.ClearCompleted(ctx, list.ID); err != nil {
		log.Fatalf("ClearCompleted: %v", err)
	}
	fmt.Println("✓ clear")

	fmt.Println("\nВсе API-операции для inline-кнопок: OK")
}
