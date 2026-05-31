// Одноразовая утилита: получить GOOGLE_REFRESH_TOKEN для .env
//
// Перед запуском в Google Cloud Console:
//   - включён Google Tasks API
//   - OAuth client (Desktop или Web)
//   - Authorized redirect URI: http://127.0.0.1:8080/oauth/callback
//
// Запуск:
//   export GOOGLE_CLIENT_ID=...
//   export GOOGLE_CLIENT_SECRET=...
//   go run ./cmd/google-auth
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	taskapi "google.golang.org/api/tasks/v1"
)

const redirectURL = "http://127.0.0.1:8080/oauth/callback"

func main() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatal("Задайте GOOGLE_CLIENT_ID и GOOGLE_CLIENT_SECRET (из .env или export)")
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{taskapi.TasksScope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	state := fmt.Sprintf("st-%d", time.Now().UnixNano())
	authURL := cfg.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			errCh <- fmt.Errorf("state mismatch")
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			http.Error(w, "OAuth error: "+errMsg, http.StatusBadRequest)
			errCh <- fmt.Errorf("oauth error: %s", errMsg)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			errCh <- fmt.Errorf("missing code")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><body><h2>Готово</h2><p>Можно закрыть вкладку и вернуться в терминал.</p></body></html>`)
		codeCh <- code
	})

	srv := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	fmt.Println("1) Откройте в браузере (войдите в нужный Google-аккаунт):")
	fmt.Println()
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("2) После подтверждения доступа скопируйте refresh token ниже в .env → GOOGLE_REFRESH_TOKEN")
	fmt.Println()

	select {
	case err := <-errCh:
		log.Fatal(err)
	case code := <-codeCh:
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tok, err := cfg.Exchange(ctx, code)
		if err != nil {
			log.Fatalf("exchange: %v", err)
		}
		if tok.RefreshToken == "" {
			log.Fatal("refresh token пустой. Удалите доступ приложения в https://myaccount.google.com/permissions и запустите снова (нужен prompt=consent).")
		}
		fmt.Println("GOOGLE_REFRESH_TOKEN=" + tok.RefreshToken)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
