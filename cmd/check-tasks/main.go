// Проверка подключения к Google Tasks (без Telegram).
//
//	go run ./cmd/check-tasks
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sergey/GoogleTaskBot/internal/config"
	"github.com/sergey/GoogleTaskBot/internal/tasks"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	client, err := tasks.NewClient(ctx, cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRefreshToken)
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	lists, err := client.ListTaskLists(ctx)
	if err != nil {
		log.Fatalf("lists: %v", err)
	}
	fmt.Fprintf(os.Stdout, "Списков: %d\n", len(lists))

	all, err := client.ListAllActiveTasks(ctx)
	if err != nil {
		log.Fatalf("tasks: %v", err)
	}
	loc := cfg.Timezone
	tasks.SortForDisplay(all, loc)

	var overdue, today int
	for _, t := range all {
		if tasks.IsOverdue(t, loc) {
			overdue++
		}
		if tasks.IsDueToday(t, loc) {
			today++
		}
	}
	fmt.Fprintf(os.Stdout, "Активных задач: %d (просрочено: %d, на сегодня: %d)\n", len(all), overdue, today)

	for i, t := range all {
		if i >= 5 {
			fmt.Fprintf(os.Stdout, "... и ещё %d\n", len(all)-5)
			break
		}
		fmt.Fprintf(os.Stdout, "  %s\n", tasks.FormatTaskLine(t, loc))
	}
	fmt.Println("OK")
}
