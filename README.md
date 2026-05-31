# Google Task Bot

Telegram-бот для личного Google Tasks: просмотр списков, добавление задач с дедлайном через кнопки, утренние сводки и напоминания.

## Возможности

- **Все задачи** — сводка с счётчиками, сортировка: просроченные → сегодня → позже → без срока
- **Просроченные** — отдельный фильтр с подсветкой 🔴
- **На сегодня** / **Списки** — просмотр по спискам Google Tasks
- **Выполненные** — последние 30, вернуть в работу или удалить
- **Карточка задачи** — выполнить, сменить срок, заметки, удалить
- **Добавление** — название → список → срок кнопками
- Утренняя рассылка: просроченные + задачи на сегодня
- За час до дедлайна и напоминания по задачам без срока
- Доступ только пользователям из `ALLOWED_USER_IDS`

Маркеры: 🔴 просрочено · 🟡 сегодня · 🔵 завтра · 🟢 позже · ⚪ без срока

## Переменные окружения

Скопируйте `.env.example` в `.env` и заполните:

| Переменная | Описание |
|------------|----------|
| `TELEGRAM_BOT_TOKEN` | Токен от [@BotFather](https://t.me/BotFather) |
| `ALLOWED_USER_IDS` | ID через запятую (например `848625725`) |
| `GOOGLE_CLIENT_ID` | OAuth Client ID |
| `GOOGLE_CLIENT_SECRET` | OAuth Client Secret |
| `GOOGLE_REFRESH_TOKEN` | Refresh token с scope Tasks |
| `TZ` | Часовой пояс (по умолчанию `Europe/Moscow`) |
| `MORNING_NOTIFY_TIME` | Время утренней сводки `HH:MM` |
| `REMINDER_INTERVAL` | Интервал напоминаний без дедлайна (например `4h`) |
| `POLL_INTERVAL` | Как часто проверять дедлайны (например `1m`) |

## Google OAuth

### Шаг 1 — Google Cloud Console

1. Откройте [Google Cloud Console](https://console.cloud.google.com/) → создайте проект (или выберите существующий).
2. **APIs & Services → Library** → найдите **Google Tasks API** → **Enable**.
3. **APIs & Services → OAuth consent screen**:
   - User type: **External** (для личного аккаунта) или Internal (Workspace).
   - Заполните название приложения, email.
   - **Test users** — добавьте свой Gmail, пока приложение в статусе Testing.
4. **APIs & Services → Credentials → Create Credentials → OAuth client ID**:
   - Application type: **Web application** (удобнее для redirect) или Desktop.
   - **Authorized redirect URIs**: `http://127.0.0.1:8080/oauth/callback`
   - Сохраните **Client ID** и **Client secret** в `.env`.

### Шаг 2 — Refresh token (утилита в репозитории)

В `.env` уже должны быть `GOOGLE_CLIENT_ID` и `GOOGLE_CLIENT_SECRET` (без refresh token).

```bash
set -a && source .env && set +a
go run ./cmd/google-auth
```

Скопируйте ссылку из терминала в браузер → войдите в Google → разрешите доступ → в терминале появится строка `GOOGLE_REFRESH_TOKEN=...` — вставьте значение в `.env`.

Если refresh token пустой: [управление доступом](https://myaccount.google.com/permissions) → удалите своё приложение → запустите `go run ./cmd/google-auth` снова.

### Альтернатива — OAuth Playground

[OAuth 2.0 Playground](https://developers.google.com/oauthplayground/) → шестерёнка → Use your own OAuth credentials → scope `https://www.googleapis.com/auth/tasks` → Authorize → Exchange → Refresh token.

## Проверка Google API

```bash
set -a && source .env && set +a
go run ./cmd/check-tasks
```

## Локальный запуск

```bash
cp .env.example .env
# отредактируйте .env
go run ./cmd/bot
```

## Docker

```bash
cp .env.example .env
docker compose build
docker compose up -d
```

## Команды бота

- `/start`, `/help` — справка и меню
- `/tasks` — все активные задачи
- `/overdue` — просроченные
- `/today` — на сегодня
- `/lists` — выбор списка
- `/completed` — выполненные
- `/add` — новая задача

Узнать свой Telegram ID: [@userinfobot](https://t.me/userinfobot).
