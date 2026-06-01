package tasks

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	taskapi "google.golang.org/api/tasks/v1"
)

type Client struct {
	svc *taskapi.Service
}

type TaskList struct {
	ID    string
	Title string
}

type Task struct {
	ID        string
	ListID    string
	ListName  string
	Title     string
	Notes     string
	Due       *time.Time
	Status    string
	Completed *time.Time
	WebLink   string
}

func NewClient(ctx context.Context, clientID, clientSecret, refreshToken string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		Scopes: []string{taskapi.TasksScope},
	}
	token := &oauth2.Token{RefreshToken: refreshToken}
	ts := cfg.TokenSource(ctx, token)

	// WithHTTPClient bypasses WithTokenSource in google.golang.org/api — attach OAuth to the transport.
	base := httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	oauthHTTP := &http.Client{
		Transport: &oauth2.Transport{
			Source: ts,
			Base:   base,
		},
		Timeout: httpClient.Timeout,
	}

	svc, err := taskapi.NewService(ctx, option.WithHTTPClient(oauthHTTP))
	if err != nil {
		return nil, fmt.Errorf("create tasks service: %w", err)
	}
	return &Client{svc: svc}, nil
}

// Ping checks credentials by refreshing the access token and listing task lists.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.ListTaskLists(ctx)
	return err
}

func (c *Client) ListTaskLists(ctx context.Context) ([]TaskList, error) {
	resp, err := c.svc.Tasklists.List().Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	lists := make([]TaskList, 0, len(resp.Items))
	for _, item := range resp.Items {
		if item.Id == "" {
			continue
		}
		lists = append(lists, TaskList{ID: item.Id, Title: item.Title})
	}
	return lists, nil
}

func (c *Client) GetTask(ctx context.Context, listID, taskID string) (*Task, error) {
	t, err := c.svc.Tasks.Get(listID, taskID).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return mapTask(t, listID, ""), nil
}

func (c *Client) ListTasks(ctx context.Context, listID string, showCompleted bool) ([]Task, error) {
	call := c.svc.Tasks.List(listID).ShowCompleted(showCompleted).ShowHidden(true)
	var all []*taskapi.Task
	for {
		resp, err := call.Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Items...)
		if resp.NextPageToken == "" {
			break
		}
		call = call.PageToken(resp.NextPageToken)
	}

	out := make([]Task, 0, len(all))
	for _, t := range all {
		if t == nil || t.Id == "" {
			continue
		}
		out = append(out, *mapTask(t, listID, ""))
	}
	return out, nil
}

func (c *Client) ListAllActiveTasks(ctx context.Context) ([]Task, error) {
	lists, err := c.ListTaskLists(ctx)
	if err != nil {
		return nil, err
	}
	var result []Task
	for _, list := range lists {
		tasks, err := c.ListTasks(ctx, list.ID, false)
		if err != nil {
			return nil, fmt.Errorf("list %q: %w", list.Title, err)
		}
		for i := range tasks {
			tasks[i].ListName = list.Title
			if tasks[i].Status != "completed" {
				result = append(result, tasks[i])
			}
		}
	}
	return result, nil
}

func (c *Client) ListCompletedTasks(ctx context.Context, limit int) ([]Task, error) {
	lists, err := c.ListTaskLists(ctx)
	if err != nil {
		return nil, err
	}
	var result []Task
	for _, list := range lists {
		tasks, err := c.ListTasks(ctx, list.ID, true)
		if err != nil {
			return nil, err
		}
		for _, t := range tasks {
			if t.Status == "completed" {
				t.ListName = list.Title
				result = append(result, t)
			}
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		ci, cj := result[i].Completed, result[j].Completed
		if ci != nil && cj != nil {
			return ci.After(*cj)
		}
		return result[i].Title < result[j].Title
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (c *Client) CreateTask(ctx context.Context, listID, title, notes string, due *time.Time) (*Task, error) {
	item := &taskapi.Task{Title: title, Notes: notes}
	if due != nil {
		item.Due = formatDue(*due)
	}
	created, err := c.svc.Tasks.Insert(listID, item).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return mapTask(created, listID, ""), nil
}

func (c *Client) UpdateTask(ctx context.Context, listID, taskID string, patch *taskapi.Task) (*Task, error) {
	updated, err := c.svc.Tasks.Patch(listID, taskID, patch).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return mapTask(updated, listID, ""), nil
}

func (c *Client) CompleteTask(ctx context.Context, listID, taskID string) error {
	_, err := c.UpdateTask(ctx, listID, taskID, &taskapi.Task{Status: "completed"})
	return err
}

func (c *Client) ReopenTask(ctx context.Context, listID, taskID string) error {
	empty := ""
	_, err := c.UpdateTask(ctx, listID, taskID, &taskapi.Task{Status: "needsAction", Completed: &empty})
	return err
}

func (c *Client) UpdateNotes(ctx context.Context, listID, taskID, notes string) error {
	_, err := c.UpdateTask(ctx, listID, taskID, &taskapi.Task{Notes: notes})
	return err
}

func (c *Client) UpdateDue(ctx context.Context, listID, taskID string, due *time.Time) error {
	patch := &taskapi.Task{}
	if due == nil {
		patch.Due = ""
	} else {
		patch.Due = formatDue(*due)
	}
	_, err := c.UpdateTask(ctx, listID, taskID, patch)
	return err
}

func (c *Client) UpdateTitle(ctx context.Context, listID, taskID, title string) error {
	_, err := c.UpdateTask(ctx, listID, taskID, &taskapi.Task{Title: title})
	return err
}

func (c *Client) DeleteTask(ctx context.Context, listID, taskID string) error {
	return c.svc.Tasks.Delete(listID, taskID).Context(ctx).Do()
}

func (c *Client) ClearCompleted(ctx context.Context, listID string) error {
	return c.svc.Tasks.Clear(listID).Context(ctx).Do()
}

func mapTask(t *taskapi.Task, listID, listName string) *Task {
	task := &Task{
		ID:       t.Id,
		ListID:   listID,
		ListName: listName,
		Title:    t.Title,
		Notes:    t.Notes,
		Status:   t.Status,
		WebLink:  t.WebViewLink,
	}
	if t.Due != "" {
		if due, err := parseDue(t.Due); err == nil {
			task.Due = &due
		}
	}
	if t.Completed != nil && *t.Completed != "" {
		if completed, err := time.Parse(time.RFC3339, *t.Completed); err == nil {
			task.Completed = &completed
		}
	}
	return task
}

func parseDue(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse due %q", s)
}

func formatDue(t time.Time) string {
	// Google Tasks keeps date only — use noon UTC to reduce timezone drift
	y, m, d := t.Date()
	return time.Date(y, m, d, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
}
