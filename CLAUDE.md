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

- **`internal/crpt/`** — HTTP client wrapping the CRPT True API v4 (`go-resty`). `auth.go` implements the two-step authentication: GET `/auth/key` → sign the challenge via `sign-service` → POST `/auth/simpleSignIn` → JWT. The JWT is stored in a cookie (`crpt_token`) and passed by handlers on each request; it is never cached server-side.
- **`internal/signer/`** — gRPC client for `sign-service`. Filters certificates from the Windows store, removing non-personal entries (Adobe CAs, UUID-named system certs). The `Signer` interface in `internal/crpt/client.go` is the seam between these two packages.
- **`internal/web/handlers/`** — Fiber HTTP handlers. Auth credentials (thumbprint, INN, МЧД) come from config defaults but can be overridden per-request via form fields.
- **`views/`** — `templ` templates. Every `*.templ` file has a corresponding generated `*_templ.go` that must be regenerated after edits (`make templ`). Never edit `*_templ.go` directly.
- **`config/`** — `cleanenv` reads YAML with ENV overrides. `MustLoad` panics on missing file. Copy `config/example.yml` → `config/config.yml` or `config/prod.yml` to get started.

### Request flow

```
Browser → Fiber (middleware/logger.go) → handler → crpt.Client → CRPT API v4
                                                  ↘ signer.Client → sign-service gRPC (Windows)
```

### Config env overrides

`SERVER_ADDR`, `CRPT_BASE_URL`, `CRPT_THUMBPRINT`, `CRPT_INN`, `CRPT_MCHD`, `SIGNER_ADDR`, `LOG_LEVEL`

---

## Documents API — True API v4

> ⚠️ Старый эндпоинт `/true-api/facade/doc` (v3) возвращает 404. Использовать **только v4**.

### Base URL pattern

```
{CRPT_BASE_URL}/api/v4/true-api/...
```

All requests require header: `Authorization: Bearer <JWT_TOKEN>` (from cookie `crpt_token`).

---

### 2.1 GET /api/v4/true-api/doc/list — список документов

**Товарная группа:** `water` (код БД: 13). Параметр `pg=water` обязателен во всех запросах.

**Query-параметры:**

| Параметр       | Тип      | Обязательный | Описание                                                                 |
|----------------|----------|--------------|--------------------------------------------------------------------------|
| `pg`           | string   | ✅            | Товарная группа: `water`                                                 |
| `input`        | bool     | ✅            | `true` — входящие документы от поставщиков                               |
| `documentType` | []string | нет          | Типы отгрузок: `LP_SHIP_GOODS`, `LP_SHIP_GOODS_CSV`, `LP_SHIP_GOODS_XML` |
| `dateFrom`     | string   | нет          | ISO 8601, напр. `2024-01-01T00:00:00.000Z`                               |
| `dateTo`       | string   | нет          | ISO 8601                                                                 |
| `limit`        | int      | нет          | Макс. 1000, по умолчанию 50                                              |

**Пример запроса:**
```bash
curl "{CRPT_BASE_URL}/api/v4/true-api/doc/list?pg=water&input=true&limit=50" \
  -H "Authorization: Bearer <TOKEN>"
```

**Ответ — Go модель (`internal/model/document.go`):**
```go
type DocListResponse struct {
    Results  []Document `json:"results"`
    NextPage bool       `json:"nextPage"`
}

type Document struct {
    Number    string `json:"number"`    // ID документа — использовать в /docs/:id
    Type      string `json:"type"`      // LP_SHIP_GOODS | LP_SHIP_GOODS_CSV | LP_SHIP_GOODS_XML
    DocDate   string `json:"docDate"`   // ISO 8601
    SenderInn string `json:"senderInn"` // ИНН поставщика
    Status    string `json:"status"`    // WAIT_ACCEPTANCE | ACCEPTED | REJECTED | ...
}
```

---

### 2.2 GET /api/v4/true-api/doc/{docId}/info — детали документа (КИ)

**Query-параметры:**

| Параметр | Тип    | Обязательный | Описание                                              |
|----------|--------|--------------|-------------------------------------------------------|
| `pg`     | string | ✅            | `water`                                               |
| `body`   | bool   | ✅            | `true` — **без этого параметра КИ не возвращаются**   |

**Пример запроса:**
```bash
curl "{CRPT_BASE_URL}/api/v4/true-api/doc/{docId}/info?pg=water&body=true" \
  -H "Authorization: Bearer <TOKEN>"
```

**Ответ — Go модель (`internal/model/document.go`):**
```go
type DocInfoResponse struct {
    Number    string  `json:"number"`
    Type      string  `json:"type"`
    DocDate   string  `json:"docDate"`
    SenderInn string  `json:"senderInn"`
    Status    string  `json:"status"`
    Body      DocBody `json:"body"`
}

type DocBody struct {
    // Для water-группы КИ находятся в products.
    // Для других форматов может быть cisesList — проверять оба поля.
    Products  []Product `json:"products"`
    CisesList []string  `json:"cisesList"`
}

type Product struct {
    Cis          string `json:"cis"`          // Код маркировки (КИ)
    // КИ могут возвращаться в нормализованном виде (без скобок AI 01/21)
    // Отображать as-is, не трансформировать.
}
```

---

## Handlers — реализация

### GET /docs

```
internal/web/handlers/documents.go → func ListDocuments(c *fiber.Ctx) error
```

1. Взять JWT из cookie `crpt_token`; если отсутствует — редирект на `/auth`.
2. Вызвать `crpt.Client.ListDocuments(ctx, token, DocListParams{PG: "water", Input: true, Limit: 50})`.
3. Передать `[]model.Document` в templ-шаблон `views/documents.templ`.
4. Таблица: `number`, `type`, `docDate`, `senderInn`, `status` + ссылка «Детали» → `/docs/{number}`.

### GET /docs/:id

```
internal/web/handlers/documents.go → func ShowDocument(c *fiber.Ctx) error
```

1. Взять `docId` из `c.Params("id")`.
2. Взять JWT из cookie `crpt_token`; если отсутствует — редирект на `/auth`.
3. Вызвать `crpt.Client.GetDocumentInfo(ctx, token, docId, "water")`.
4. Парсить `body.products` (primary) или `body.cisesList` (fallback).
5. Передать метаданные + список КИ в `views/document_detail.templ`.

### crpt.Client — новые методы

```go
// internal/crpt/documents.go

func (c *Client) ListDocuments(ctx context.Context, token string, p DocListParams) (*model.DocListResponse, error)
// GET /api/v4/true-api/doc/list?pg={p.PG}&input={p.Input}&limit={p.Limit}&...

func (c *Client) GetDocumentInfo(ctx context.Context, token, docID, pg string) (*model.DocInfoResponse, error)
// GET /api/v4/true-api/doc/{docID}/info?pg={pg}&body=true
```

**Удалить старый метод** (если существует): любой вызов к `/true-api/facade/doc` или эндпоинтам v3.

---

## Constraints

- `sign-service` only runs on Windows — gRPC calls will fail locally unless tunnelled or mocked.
- JWT is not cached; auth must be repeated after expiry (~10 h).
- **True API v4 only** — пути без `/api/v4/` prefix возвращают 404.
- Параметр `pg=water` обязателен в каждом запросе к doc API.
- Параметр `body=true` обязателен для `/doc/{id}/info` — иначе КИ не возвращаются.

## Обработка ошибок

| HTTP | Действие |
|---|---|
| 401 | Токен истёк — редирект на `/auth` для повторной авторизации |
| 400 | Логировать тело ответа (`error_message`), вернуть ошибку пользователю |
| 403 | У сертификата нет доступа к методу — показать сообщение |
| 404 | Неверный путь API — залогировать полный URL запроса для диагностики |
| 5xx | Логировать, показать «сервис временно недоступен» |

## Что НЕ реализовывать на этом этапе

- Приёмка / отклонение документа (accept/reject actions)
- Кэширование токена и его автообновление
- Пагинация через UI — первые 50 записей, `limit` хардкодить в клиенте
- Экспорт в Excel/CSV
- Фильтрация по `documentStatus` по умолчанию — показывать все статусы
