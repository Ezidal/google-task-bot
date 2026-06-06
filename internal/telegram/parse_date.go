package telegram

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDueInput understands DD.MM, DD.MM.YYYY, +N, +Nd, +Nw.
func ParseDueInput(raw string, loc *time.Location) (*time.Time, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return nil, fmt.Errorf("пустая дата")
	}

	now := time.Now().In(loc)
	if strings.HasPrefix(s, "+") {
		return parseRelativeDue(s[1:], now, loc)
	}

	sep := "."
	if strings.Contains(s, "/") {
		sep = "/"
	}
	parts := strings.Split(s, sep)
	var year int
	switch len(parts) {
	case 2:
		year = now.Year()
	case 3:
		y, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("неверный год")
		}
		if y < 100 {
			y += 2000
		}
		year = y
	default:
		return nil, fmt.Errorf("формат: ДД.ММ или ДД.ММ.ГГГГ или +3d")
	}
	d, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("неверный день")
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("неверный месяц")
	}
	t := time.Date(year, time.Month(m), d, 12, 0, 0, 0, loc)
	if t.Month() != time.Month(m) || t.Day() != d {
		return nil, fmt.Errorf("неверная дата")
	}
	utc := t.UTC()
	return &utc, nil
}

func parseRelativeDue(s string, now time.Time, loc *time.Location) (*time.Time, error) {
	days := 0
	if s == "" {
		return nil, fmt.Errorf("укажите смещение, например +3d")
	}
	unit := s[len(s)-1]
	numPart := s
	if (unit == 'd' || unit == 'w') && len(s) > 1 {
		numPart = s[:len(s)-1]
		n, err := strconv.Atoi(numPart)
		if err != nil {
			return nil, fmt.Errorf("неверное смещение")
		}
		if unit == 'w' {
			days = n * 7
		} else {
			days = n
		}
	} else {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("неверное смещение")
		}
		days = n
	}
	d := now.AddDate(0, 0, days)
	t := time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, loc)
	utc := t.UTC()
	return &utc, nil
}
