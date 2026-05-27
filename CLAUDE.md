# CLAUDE.md — lrclib CLI

> Операционные инструкции для Claude Code. Загружается в каждую сессию — только то, что Claude не выведет из кода сам.

---

## TL;DR — прочти первым

1. **Ветки обязательны** — любая задача = своя ветка от `master`, коммит после каждой микрозадачи
2. **Перед PR** — `make fmt && make lint && make test` должны быть зелёными
3. **CGO_ENABLED=0 везде** — без исключений, иначе GoReleaser не соберёт кросс-платформу
4. **Кэш — degraded, не fatal** — ошибка кэша пишется как WARN и не роняет CLI
5. **`context.Context` — первым аргументом** в любой функции, работающей с IO

---

## Обзор проекта

**lrclib** — кросс-платформенная CLI-утилита на Go для поиска и скачивания текстов песен с [lrclib.net](https://lrclib.net) и сохранения в `.lrc`-файлы. Включает TUI-интерфейс для интерактивного поиска.

**Стек:** Go 1.22+, Cobra (CLI), Bubble Tea (TUI), slog (логи), modernc.org/sqlite (кэш, без CGO), GoReleaser, GitHub Actions.

---

## Структура проекта

```
lrclib/
├── cmd/lrclib/          # main.go — только DI-инициализация и запуск
├── internal/
│   ├── api/             # HTTP-клиент lrclib.net
│   ├── cache/           # SQLite-кэш (modernc.org/sqlite)
│   ├── config/          # Конфиг (XDG/APPDATA), env-переменные
│   ├── errs/            # Типы ошибок, exit codes (внутренние)
│   ├── lrc/             # Парсинг и генерация .lrc-формата
│   ├── tui/             # Bubble Tea TUI
│   └── usecase/         # Бизнес-логика (не знает про Cobra и Bubble Tea)
├── build/               # .goreleaser.yml, Dockerfile, скрипты CI
├── configs/             # Шаблоны конфигов
├── docs/                # Документация, в т.ч. lrclib API-справочник
├── scripts/             # Вспомогательные shell-скрипты
├── .github/
│   ├── workflows/       # ci.yml, release.yml
│   ├── PULL_REQUEST_TEMPLATE.md
│   ├── ISSUE_TEMPLATE/
│   └── dependabot.yml
├── .golangci.yml
├── .lefthook.yml
├── Makefile
└── CLAUDE.md
```

**Правила:**
- `cmd/lrclib/main.go` — только wire-up зависимостей и `rootCmd.Execute()`
- Всё приватное — в `internal/`, публичного `pkg/` нет (это бинарь, не библиотека)
- Зависимости между слоями — только через интерфейсы, определённые на стороне потребителя
- `internal/usecase/` не импортирует `internal/api`, `internal/cache` напрямую — только интерфейсы

---

## Слои архитектуры

```
cmd/                        ← composition root, DI
  ↓
internal/tui  (presentation)
  ↓
internal/usecase            ← бизнес-логика, не знает о внешнем мире
  ↓  (через интерфейсы)
internal/api    internal/cache    internal/config   ← infrastructure
  ↓
internal/lrc                ← domain: LRC-формат, entities
  ↓
internal/errs               ← базовые типы ошибок
```

---

## Рабочий процесс

### Декомпозиция

```
Эпик / Фича
└── Подзадача → отдельная ветка (feature/<name>, fix/<name>, chore/<name>)
    └── Микрозадача → 1 коммит в ветке подзадачи
```

Подзадача завершена → PR → code review → merge в `master`. В `master` — никогда напрямую.

### Conventional Commits

```
<type>(<scope>): <описание>
```

Типы: `feat` `fix` `refactor` `test` `docs` `ci` `chore` `perf`

```
feat(api): add synced lyrics endpoint
fix(cache): handle WAL lock on concurrent writes
perf(http): enable connection pooling with MaxIdleConnsPerHost
```

### Definition of Done (перед каждым PR)

- [ ] `make fmt` — без diff
- [ ] `make lint` — без ошибок
- [ ] `make test` — все тесты зелёные
- [ ] Покрытие новой логики тестами ≥ 70%
- [ ] GoDoc-комментарии на новых экспортируемых символах
- [ ] `CHANGELOG.md` обновлён (для пользовательских фич)

### Когда спрашивать пользователя

**Спрашивай** перед: добавлением новой зависимости, изменением публичного CLI-интерфейса (флаги, команды), удалением файлов, изменением CI-воркфлоу.

**Действуй сам:** рефакторинг внутри пакета, добавление тестов, исправление линтера, обновление документации.

---

## Сборка и запуск

```bash
make build          # go build ./cmd/lrclib
make run            # go run ./cmd/lrclib
make test           # go test ./... (unit; integration исключены build-тегом)
make test-int       # go test -tags=integration ./...
make lint           # golangci-lint run
make fmt            # gofumpt -w .
make clean          # rm -rf dist/ tmp/
make release-dry    # goreleaser release --snapshot --clean
```

**CGO_ENABLED=0 везде.** Версия вшита через ldflags:
```
-ldflags "-X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}"
```

Платформы GoReleaser: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`.

---

## Стиль кода

- Форматтер: `gofumpt` (строже чем `gofmt`)
- Линтер: `golangci-lint` с конфигом `.golangci.yml`
- Тесты: table-driven, моки через интерфейсы — не конкретные структуры
- Интерфейсы определяются на стороне **потребителя**, не реализации
- Экспортируемые символы — с GoDoc-комментариями

---

## lrclib API (`internal/api/`)

**Документация:** https://lrclib.net/docs

### Верифицированные endpoints

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/api/get` | Получить lyrics по метаданным трека |
| `GET` | `/api/get/:id` | Получить lyrics по числовому ID |
| `GET` | `/api/search` | Поиск треков |

**GET /api/get** — параметры: `track_name`, `artist_name`, `album_name`, `duration` (сек, float).

**GET /api/search** — параметр: `q` (свободный текст).

**Структура ответа** (одиночный объект или массив):
```json
{
  "id": 3718600,
  "name": "Creep",
  "trackName": "Creep",
  "artistName": "Radiohead",
  "albumName": "Pablo Honey",
  "duration": 236.0,
  "instrumental": false,
  "plainLyrics": "...",
  "syncedLyrics": "[00:00.00] ...",
  "lyricsfile": null
}
```

### HTTP-клиент

- `User-Agent: lrclib-cli/<version> (github.com/your-org/lrclib)`
- Таймауты: connect 5s, read 15s, idle 90s
- `http.Transport`: `MaxIdleConnsPerHost: 10`, `DisableCompression: false`
- Retry: exponential backoff с jitter, максимум 3 попытки
- Уважать заголовок `Retry-After` при статусе 429

### Fallback-логика (graceful degradation)

```
1. GET /api/get (по метаданным) → synced lyrics
2. Нет synced → использовать plainLyrics из того же ответа
3. Нет ответа → GET /api/search → взять первый результат
4. API недоступен → проверить локальный кэш
5. Кэш пуст → user-friendly ошибка (exit code 1)
```

---

## LRC-формат (`internal/lrc/`)

- Временны́е метки: `[mm:ss.xx]` (синхронизированные)
- Мета-теги: `[ti:]`, `[ar:]`, `[al:]`, `[length:]`, `[by:]`
- Парсить оба поля ответа: `syncedLyrics` и `plainLyrics`
- Golden-файлы для тестов в `internal/lrc/testdata/`
- Plain lyrics (без меток) — валидный .lrc, если synced недоступны

---

## Кэширование (`internal/cache/`)

**Backend:** `modernc.org/sqlite` — pure Go, без CGO, работает везде.

**Расположение:**
- Linux/macOS: `os.UserCacheDir()/lrclib/cache.db` → `~/.cache/lrclib/`
- Windows: `os.UserCacheDir()/lrclib/cache.db` → `%LOCALAPPDATA%\lrclib\`

**Никогда не хардкодить пути** — только `os.UserCacheDir()`, `os.UserConfigDir()`.

**Схема:**
```sql
CREATE TABLE IF NOT EXISTS lyrics_cache (
    track_id   TEXT PRIMARY KEY,
    data       BLOB    NOT NULL,
    cached_at  INTEGER NOT NULL,
    ttl        INTEGER NOT NULL DEFAULT 604800  -- 7 дней в секундах
);
```

**PRAGMA при открытии БД (обязательно):**
```sql
PRAGMA journal_mode=WAL;       -- параллельные read/write без блокировок
PRAGMA busy_timeout=5000;      -- ждать 5с на locked, не падать сразу
PRAGMA synchronous=NORMAL;     -- баланс скорости и надёжности
PRAGMA foreign_keys=ON;
```

**Правила:**
- Запись: `INSERT OR REPLACE` (UPSERT)
- TTL по умолчанию: 7 дней
- Команды: `lrclib cache clear`, `lrclib cache clear --all`, `lrclib cache stats`
- Ошибка кэша → логировать как WARN, продолжать без кэша

---

## Логирование

**Библиотека:** `log/slog` (стандартная библиотека Go).

| Флаг | Поведение |
|------|-----------|
| по умолчанию | `TextHandler`, уровень INFO |
| `--log-format=json` | `JSONHandler` |
| `--log-level=debug` | показать DEBUG |
| `LRCLIB_LOG_LEVEL=debug` | то же через env |

```go
slog.Error("api request failed",
    "url", url,
    "status", resp.StatusCode,
    "attempt", attempt,
    "err", err,
)
```

Sensitive данные (токены, приватные пути) — не логировать никогда.

---

## Обработка ошибок (`internal/errs/`)

### Типы

```go
type Kind int
const (
    KindNotFound    Kind = iota // HTTP 404, нет в кэше
    KindRateLimited             // HTTP 429
    KindNetwork                 // таймаут, DNS, conn refused
    KindCanceled                // context.Canceled / DeadlineExceeded
    KindInternal                // непредвиденная ошибка
    KindBadInput                // неверные аргументы пользователя
)
```

### Exit codes

| Ситуация | Code |
|----------|------|
| Успех | 0 |
| Не найдено | 1 |
| Ошибка сети | 2 |
| Rate limited | 3 |
| Внутренняя ошибка | 4 |
| Неверные аргументы | 5 |

`os.Exit` — только в `main.go`. `panic` — запрещён в production-коде.

### Вывод ошибок

```
# Терминал (по умолчанию)
Error: lyrics not found for "Radiohead – Creep"

# JSON-режим (--output=json)
{"error": "lyrics not found", "kind": "not_found", "track": "Radiohead – Creep"}
```

Использовать `errors.Is` / `errors.As` — не сравнивать строки ошибок.

---

## Тестирование

| Вид | Тег | Где |
|-----|-----|-----|
| Unit | _(нет)_ | рядом с кодом, `_test.go` |
| Integration | `//go:build integration` | реальный SQLite, `httptest.Server` |

- Моки HTTP: `net/http/httptest.NewServer`
- Моки зависимостей: интерфейсы, не конкретные структуры
- Реальные запросы к lrclib.net в unit-тестах — **запрещены**
- Golden-файлы для .lrc-вывода в `testdata/`

---

## CI/CD

### `ci.yml` — каждый PR и push в `master`

```
go vet ./...
golangci-lint run
go test ./...
go test -tags=integration ./...
go build -o /dev/null ./cmd/lrclib   # проверка компиляции
```

### `release.yml` — тег `v*.*.*`

```
goreleaser release --clean
→ .tar.gz / .zip для каждой платформы
→ checksums.txt
→ GitHub Release с changelog из Conventional Commits
```

### Pre-commit (`.lefthook.yml`)

```yaml
pre-commit:
  commands:
    fmt:   run: gofumpt -l .
    vet:   run: go vet ./...
    lint:  run: golangci-lint run --fast
```

### Версионирование

SemVer строго: `vMAJOR.MINOR.PATCH`. Pre-release: `v1.0.0-rc.1`.
`CHANGELOG.md` по формату [Keep a Changelog](https://keepachangelog.com).

---

## TUI (`internal/tui/`)

- **Библиотека:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) + `bubbles` + `lipgloss`
- Запуск: `lrclib search --tui` или `lrclib tui`
- Строгое разделение: `Model`, `Update`, `View` — по TEA-паттерну
- Долгие операции (HTTP, файлы) — через `tea.Cmd`, не блокировать `Update`
- Глобальное состояние в TUI-пакете — запрещено

---

## Конфигурация (`internal/config/`)

- Linux/macOS: `os.UserConfigDir()/lrclib/config.toml`
- Windows: `os.UserConfigDir()/lrclib/config.toml` → `%APPDATA%\lrclib\`
- Приоритет: флаги CLI > env (`LRCLIB_*`) > config.toml > defaults
- Пример: `LRCLIB_CACHE_TTL=86400`, `LRCLIB_LOG_LEVEL=debug`

---

## Чего НЕ делать

| Запрет | Причина |
|--------|---------|
| `CGO_ENABLED=1` | Ломает кросс-компиляцию GoReleaser |
| `os.Exit` вне `main.go` | Невозможно перехватить, тесты падают |
| `panic` в production | Некорректное завершение, нет cleanup |
| Игнорировать ошибки (`_ =`) | Кроме кэша — там WARN и продолжить |
| Реальные HTTP в unit-тестах | Flaky тесты, зависимость от сети |
| Хардкодить пути | Использовать `os.UserCacheDir()` / `os.UserConfigDir()` |
| Глобальные переменные | Кроме `version`, `commit`, `date` (ldflags) |
| Коммит напрямую в `master` | Только через PR |
| Логировать ошибки кэша как ERROR | Только WARN — кэш degraded, не fatal |
