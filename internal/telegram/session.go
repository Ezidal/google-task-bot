package telegram

import (
	"sync"
	"time"

	"github.com/sergey/GoogleTaskBot/internal/tasks"
)

type pendingAction int

const (
	pendingNone pendingAction = iota
	pendingTitle
	pendingListPick
	pendingNotesCreate
	pendingNotes
	pendingRename
	pendingCustomDueCreate
	pendingCustomDueEdit
)

type viewKind string

const (
	viewAll       viewKind = "all"
	viewOverdue   viewKind = "overdue"
	viewToday     viewKind = "today"
	viewTomorrow  viewKind = "tomorrow"
	viewSearch    viewKind = "search"
	viewList      viewKind = "list"
	viewCompleted viewKind = "completed"
)

type viewState struct {
	Kind     viewKind
	ListID   string
	ListName string
	Tasks    []tasks.Task
	Page     int
}

type session struct {
	action       pendingAction
	title        string
	notes        string
	listID       string
	listName     string
	dueTime      *time.Time
	searchQuery  string
	taskIndex    int
	view         *viewState
	pickerLists  []tasks.TaskList
}

type SessionStore struct {
	mu   sync.Mutex
	data map[int64]*session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{data: make(map[int64]*session)}
}

func (s *SessionStore) Reset(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, userID)
}

func (s *SessionStore) ClearPending(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess := s.data[userID]; sess != nil {
		sess.action = pendingNone
		sess.taskIndex = -1
	}
}

func (s *SessionStore) SetPickerLists(userID int64, lists []tasks.TaskList) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.get(userID)
	sess.pickerLists = append([]tasks.TaskList(nil), lists...)
}

func (s *SessionStore) PickerListAt(userID int64, index int) (tasks.TaskList, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.data[userID]
	if sess == nil || index < 0 || index >= len(sess.pickerLists) {
		return tasks.TaskList{}, false
	}
	return sess.pickerLists[index], true
}

func (s *SessionStore) UpdateTaskInView(userID int64, updated tasks.Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.data[userID]
	if sess == nil || sess.view == nil {
		return
	}
	for i := range sess.view.Tasks {
		if sess.view.Tasks[i].ID == updated.ID && sess.view.Tasks[i].ListID == updated.ListID {
			updated.ListName = sess.view.Tasks[i].ListName
			sess.view.Tasks[i] = updated
			sess.taskIndex = i
			return
		}
	}
}

func (s *SessionStore) get(userID int64) *session {
	if s.data[userID] == nil {
		s.data[userID] = &session{}
	}
	return s.data[userID]
}

func (s *SessionStore) StartAdd(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[userID] = &session{action: pendingTitle}
}

func (s *SessionStore) StartAddWithTitle(userID int64, title string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[userID] = &session{action: pendingListPick, title: title}
}

func (s *SessionStore) SetDueTime(userID int64, due *time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.get(userID)
	if due == nil {
		sess.dueTime = nil
		return
	}
	cp := *due
	sess.dueTime = &cp
}

func (s *SessionStore) DueTime(userID int64) *time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.data[userID]
	if sess == nil || sess.dueTime == nil {
		return nil
	}
	cp := *sess.dueTime
	return &cp
}

func (s *SessionStore) SetSearchQuery(userID int64, q string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.get(userID)
	sess.searchQuery = q
}

func (s *SessionStore) SearchQuery(userID int64) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.data[userID]
	if sess == nil {
		return ""
	}
	return sess.searchQuery
}

func (s *SessionStore) SetTitle(userID int64, title string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.get(userID)
	sess.action = pendingListPick
	sess.title = title
}

func (s *SessionStore) SetList(userID int64, listID, listName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.get(userID)
	sess.listID = listID
	sess.listName = listName
}

func (s *SessionStore) SetView(userID int64, v *viewState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.get(userID)
	sess.view = v
}

func (s *SessionStore) View(userID int64) *viewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.data[userID]
	if sess == nil || sess.view == nil {
		return nil
	}
	cp := *sess.view
	cp.Tasks = append([]tasks.Task(nil), sess.view.Tasks...)
	return &cp
}

func (s *SessionStore) TaskAt(userID int64, index int) (tasks.Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.data[userID]
	if sess == nil || sess.view == nil || index < 0 || index >= len(sess.view.Tasks) {
		return tasks.Task{}, false
	}
	return sess.view.Tasks[index], true
}

func (s *SessionStore) SetViewPage(userID int64, page int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[userID] != nil && s.data[userID].view != nil {
		s.data[userID].view.Page = page
	}
}

func (s *SessionStore) Snapshot(userID int64) (action pendingAction, title, listID, listName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.data[userID]
	if sess == nil {
		return pendingNone, "", "", ""
	}
	return sess.action, sess.title, sess.listID, sess.listName
}

func (s *SessionStore) StartNotesEdit(userID int64, taskIndex int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.get(userID)
	sess.action = pendingNotes
	sess.taskIndex = taskIndex
}

func (s *SessionStore) StartNotesCreate(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.get(userID)
	sess.action = pendingNotesCreate
}

func (s *SessionStore) StartRename(userID int64, taskIndex int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.get(userID)
	sess.action = pendingRename
	sess.taskIndex = taskIndex
}

func (s *SessionStore) StartCustomDueCreate(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.get(userID)
	sess.action = pendingCustomDueCreate
}

func (s *SessionStore) StartCustomDueEdit(userID int64, taskIndex int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.get(userID)
	sess.action = pendingCustomDueEdit
	sess.taskIndex = taskIndex
}

func (s *SessionStore) PendingTaskIndex(userID int64) (pendingAction, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.data[userID]
	if sess == nil {
		return pendingNone, -1
	}
	return sess.action, sess.taskIndex
}

func (s *SessionStore) SetTaskIndex(userID int64, index int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.get(userID)
	sess.taskIndex = index
}

func (s *SessionStore) CurrentTask(userID int64) (tasks.Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.data[userID]
	if sess == nil {
		return tasks.Task{}, false
	}
	if sess.view != nil && sess.taskIndex >= 0 && sess.taskIndex < len(sess.view.Tasks) {
		return sess.view.Tasks[sess.taskIndex], true
	}
	return tasks.Task{}, false
}

func findTaskIndex(v *viewState, taskID string) int {
	if v == nil {
		return 0
	}
	for i, t := range v.Tasks {
		if t.ID == taskID {
			return i
		}
	}
	return 0
}

func DueFromPreset(preset string, loc *time.Location) *time.Time {
	now := time.Now().In(loc)
	var d time.Time
	switch preset {
	case "today":
		d = now
	case "tomorrow":
		d = now.AddDate(0, 0, 1)
	case "week":
		d = now.AddDate(0, 0, 7)
	default:
		return nil
	}
	t := time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, loc)
	utc := t.UTC()
	return &utc
}
