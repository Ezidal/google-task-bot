package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	TelegramToken      string
	AllowedUserIDs     map[int64]struct{}
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRefreshToken string
	HTTPProxy          string
	Timezone           *time.Location
	MorningNotifyTime  string // HH:MM
	ReminderInterval   time.Duration
	PollInterval       time.Duration
}

func Load() (*Config, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	allowed := parseAllowedIDs(os.Getenv("ALLOWED_USER_IDS"))
	if len(allowed) == 0 {
		return nil, fmt.Errorf("ALLOWED_USER_IDS must contain at least one user id")
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	refreshToken := os.Getenv("GOOGLE_REFRESH_TOKEN")
	if clientID == "" || clientSecret == "" || refreshToken == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET and GOOGLE_REFRESH_TOKEN are required")
	}

	tzName := envOr("TZ", "Europe/Moscow")
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("invalid TZ %q: %w", tzName, err)
	}

	morning := envOr("MORNING_NOTIFY_TIME", "08:00")
	if _, err := time.Parse("15:04", morning); err != nil {
		return nil, fmt.Errorf("invalid MORNING_NOTIFY_TIME %q (use HH:MM): %w", morning, err)
	}

	reminderInterval, err := time.ParseDuration(envOr("REMINDER_INTERVAL", "4h"))
	if err != nil {
		return nil, fmt.Errorf("invalid REMINDER_INTERVAL: %w", err)
	}

	pollInterval, err := time.ParseDuration(envOr("POLL_INTERVAL", "1m"))
	if err != nil {
		return nil, fmt.Errorf("invalid POLL_INTERVAL: %w", err)
	}

	proxy := os.Getenv("HTTP_PROXY")
	if proxy == "" {
		proxy = os.Getenv("HTTPS_PROXY")
	}

	return &Config{
		TelegramToken:      token,
		AllowedUserIDs:     allowed,
		GoogleClientID:     clientID,
		GoogleClientSecret: clientSecret,
		GoogleRefreshToken: refreshToken,
		HTTPProxy:          proxy,
		Timezone:           loc,
		MorningNotifyTime:  morning,
		ReminderInterval:   reminderInterval,
		PollInterval:       pollInterval,
	}, nil
}

func parseAllowedIDs(raw string) map[int64]struct{} {
	out := make(map[int64]struct{})
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			continue
		}
		out[id] = struct{}{}
	}
	return out
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
