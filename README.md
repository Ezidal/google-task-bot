# Google Task Bot

Telegram-бот для личного Google Tasks: просмотр списков, добавление задач с дедлайном через кнопки, утренние сводки и напоминания.

## Возможности

- **Все задачи** — сводка с счётчиками, сортировка: просроченные → сегодня → позже → без срока
- **Просроченные** / **На сегодня** / **На завтра** — отдельные фильтры
- **Поиск** — `/find текст` по названию, заметкам и спискам
- **Списки** — просмотр по спискам Google Tasks
- **Выполненные** — последние 30, вернуть в работу или удалить
- **Карточка задачи** — выполнить, срок, переименовать, заметки, ссылка на Google, удалить
- **Добавление** — `/add` или `/add название` → список → срок (в т.ч. произвольная дата) → заметка
- Утренняя рассылка: просроченные + задачи на сегодня
- Напоминание в день дедлайна (`DEADLINE_NOTIFY_TIME`) и по задачам без срока
- Кнопки в уведомлениях: выполнить, открыть, отложить (+1ч / завтра)
- Состояние уведомлений в SQLite (без дублей после рестарта)
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
| `DEADLINE_NOTIFY_TIME` | Время напоминания в день дедлайна `HH:MM` (по умолчанию `09:00`) |
| `REMINDER_INTERVAL` | Минимальный интервал между напоминаниями без дедлайна (например `4h`) |
| `NODUE_MAX_PER_DAY` | Макс. напоминаний без дедлайна на задачу в сутки (по умолчанию `1`) |
| `POLL_INTERVAL` | Как часто проверять дедлайны (например `1m`) |
| `NOTIFY_DB_PATH` | Путь к SQLite с историей уведомлений (по умолчанию `./data/notify.db`) |

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

### Обновление на сервере

На сервере в каталоге проекта (где лежат `docker-compose.yml` и `.env`):

```bash
make deploy
```

Полная пересборка без кэша Docker:

```bash
make deploy-fresh
```

Проверить логи:

```bash
make logs
```

Данные уведомлений (`./data/notify.db`) сохраняются в volume — при пересборке не теряются.

Без Docker:

```bash
git pull
go build -o bot ./cmd/bot
# перезапустите systemd/supervisor или вручную остановите старый процесс и запустите ./bot
```

## Команды бота

- `/start`, `/help` — справка и меню
- `/tasks` — все активные задачи
- `/overdue` — просроченные
- `/today` — на сегодня
- `/tomorrow` — на завтра
- `/find текст` — поиск
- `/lists` — выбор списка
- `/completed` — выполненные
- `/add` или `/add название` — новая задача

Произвольная дата при выборе срока: `15.06`, `15.06.2026`, `+3d`, `+1w`.

Узнать свой Telegram ID: [@userinfobot](https://t.me/userinfobot).
