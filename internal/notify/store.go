package notify

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store persists notification state across restarts.
type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS notifications (
			key TEXT PRIMARY KEY,
			sent_at INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS snooze (
			key TEXT PRIMARY KEY,
			until_ts INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS task_refs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			list_id TEXT NOT NULL,
			task_id TEXT NOT NULL,
			UNIQUE(list_id, task_id)
		);
	`)
	return err
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Mark(key string) {
	_, _ = s.db.Exec(
		`INSERT INTO notifications(key, sent_at) VALUES(?, ?)
		 ON CONFLICT(key) DO UPDATE SET sent_at=excluded.sent_at`,
		key, time.Now().Unix(),
	)
}

func (s *Store) WasSent(key string, within time.Duration) bool {
	var sentAt int64
	err := s.db.QueryRow(`SELECT sent_at FROM notifications WHERE key=?`, key).Scan(&sentAt)
	if err != nil {
		return false
	}
	return time.Since(time.Unix(sentAt, 0)) < within
}

func (s *Store) SentCountToday(key string, loc *time.Location) int {
	start := startOfDay(time.Now().In(loc), loc).Unix()
	var n int
	_ = s.db.QueryRow(
		`SELECT COUNT(*) FROM notifications WHERE key=? AND sent_at>=?`,
		key, start,
	).Scan(&n)
	return n
}

func (s *Store) ShouldRemindNoDue(key string, interval time.Duration, loc *time.Location, maxPerDay int) bool {
	if s.IsSnoozed(key) {
		return false
	}
	if maxPerDay > 0 && s.SentCountToday(key, loc) >= maxPerDay {
		return false
	}
	var sentAt int64
	err := s.db.QueryRow(`SELECT sent_at FROM notifications WHERE key=?`, key).Scan(&sentAt)
	if err != nil {
		return true
	}
	return time.Since(time.Unix(sentAt, 0)) >= interval
}

func (s *Store) Snooze(key string, until time.Time) {
	_, _ = s.db.Exec(
		`INSERT INTO snooze(key, until_ts) VALUES(?, ?)
		 ON CONFLICT(key) DO UPDATE SET until_ts=excluded.until_ts`,
		key, until.Unix(),
	)
}

func (s *Store) SnoozeRef(refID int64, until time.Time) {
	s.Snooze(fmt.Sprintf("snooze:%d", refID), until)
}

func (s *Store) IsRefSnoozed(refID int64) bool {
	return s.IsSnoozed(fmt.Sprintf("snooze:%d", refID))
}

func (s *Store) IsSnoozed(key string) bool {
	var until int64
	err := s.db.QueryRow(`SELECT until_ts FROM snooze WHERE key=?`, key).Scan(&until)
	if err != nil {
		return false
	}
	if time.Now().Unix() >= until {
		_, _ = s.db.Exec(`DELETE FROM snooze WHERE key=?`, key)
		return false
	}
	return true
}

func (s *Store) RefTask(listID, taskID string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT OR IGNORE INTO task_refs(list_id, task_id) VALUES(?, ?)`,
		listID, taskID,
	)
	if err != nil {
		return 0, err
	}
	if id, err := res.LastInsertId(); err == nil && id > 0 {
		return id, nil
	}
	var id int64
	err = s.db.QueryRow(
		`SELECT id FROM task_refs WHERE list_id=? AND task_id=?`,
		listID, taskID,
	).Scan(&id)
	return id, err
}

func (s *Store) TaskFromRef(id int64) (listID, taskID string, ok bool) {
	err := s.db.QueryRow(
		`SELECT list_id, task_id FROM task_refs WHERE id=?`, id,
	).Scan(&listID, &taskID)
	return listID, taskID, err == nil
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}
