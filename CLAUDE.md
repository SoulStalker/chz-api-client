# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make templ   # Regenerate Go code from *.templ files (must run after any views/*.templ change)
make build   # templ + go build → ./bin/edo-client
make run     # build + run with config/prod.yml
make lint    # golangci-lint run ./...
make explore # run cmd/crpt-explore debug utility
```

Run the server manually with a custom config:
```bash
./bin/edo-client --config config/config.yml
```

There are no automated tests in the project.

## Architecture

Go web service integrating with the Russian product labelling system (Честный знак / CRPT). Cryptographic signing is **delegated to an external `sign-service` gRPC server** that runs on Windows and accesses the Windows Certificate Store with GOST algorithms — this service cannot run on macOS/Linux.

### Key layers

- **`internal/crpt/`** — HTTP client wrapping the CRPT True API (`go-resty`). `auth.go` implements the two-step authentication: GET `/auth/key` → sign the challenge via `sign-service` → POST `/auth/simpleSignIn` → JWT. The JWT is stored in a cookie (`crpt_token`) and passed by handlers on each request; it is never cached server-side.
- **`internal/signer/`** — gRPC client for `sign-service`. Filters certificates from the Windows store, removing non-personal entries (Adobe CAs, UUID-named system certs). The `Signer` interface in `internal/crpt/client.go` is the seam between these two packages.
- **`internal/web/handlers/`** — Fiber HTTP handlers. Auth credentials (thumbprint, INN, МЧД) come from config defaults but can be overridden per-request via form fields.
- **`views/`** — `templ` templates. Every `*.templ` file has a corresponding generated `*_templ.go` that must be regenerated after edits (`make templ`). Never edit `*_templ.go` directly.
- **`config/`** — `cleanenv` reads YAML with ENV overrides. `MustLoad` panics on missing file. Copy `config/example.yml` → `config/config.yml` or `config/prod.yml` to get started.

### Request flow

```
Browser → Fiber (middleware/logger.go) → handler → crpt.Client → CRPT API
                                                  ↘ signer.Client → sign-service gRPC (Windows)
```

### Config env overrides

`SERVER_ADDR`, `CRPT_BASE_URL`, `CRPT_THUMBPRINT`, `CRPT_INN`, `CRPT_MCHD`, `SIGNER_ADDR`, `LOG_LEVEL`

## Constraints

- `sign-service` only runs on Windows — gRPC calls will fail locally unless tunnelled or mocked.
- JWT is not cached; auth must be repeated after expiry (~10 h).
- Pagination is not implemented in the UI; `/docs` always returns page 0, 50 records.
- The following features are explicitly out of scope for this stage: document detail view, accept/reject actions, token caching, UI pagination, Excel/CSV export.

## Обработка ошибок

| HTTP | Действие |
|---|---|
| 401 | Токен истёк — редирект на `/` для повторной авторизации |
| 400 | Логировать тело ответа, вернуть ошибку пользователю |
| 403 | У сертификата нет доступа к методу — показать сообщение |
| 5xx | Логировать, показать «сервис временно недоступен» |

## Что НЕ реализовывать на этом этапе

- Просмотр содержимого конкретного документа (детали марок)
- Приёмка / отклонение документа
- Кэширование токена и его автообновление
- Пагинация через UI (только первая страница)
- Экспорт в Excel/CSV

# Problem 1
crpt facade/doc: status 404: {"error_message":"Метод с указанным URL не найден"}

вроде должно быть так: 
6.1. Метод получения списка загруженных документов в ГИС МТ
URL: /api/v4/true-api/doc/list
Тип приватности: приватный
Метод: GET
Запрос:
curl "<url стенда v4>/doc/list?pg=lp"
-H "accept: */*"
-H "Authorization: Bearer <ТОКЕН>"
Параметры запроса в pdf файле в папке doc:
