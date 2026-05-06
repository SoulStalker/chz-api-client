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

### Routes

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/` | — | Редирект на `/certs` |
| GET | `/certs` | `handlers.ListCerts` | Список сертификатов из sign-service |
| GET | `/auth` | `handlers.AuthForm` | Форма авторизации |
| POST | `/auth` | `handlers.Auth` | Авторизация → JWT в cookie `crpt_token` |
| GET | `/docs/incoming` | `handlers.ListDocuments` | Входящие документы (`input=true`) |
| GET | `/docs/outgoing` | `handlers.ListDocuments` | Исходящие документы (`input=false`) |
| GET | `/docs/:id` | `handlers.ShowDocument` | Детали документа + список КИ |

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

Rate limit: **50 req/s** — не превышать при навигации.

---

### GET /api/v4/true-api/doc/list — список документов

**Товарная группа:** `water` (код БД: 13). Параметр `pg=water` обязателен.

**Query-параметры:**

| Параметр             | Тип    | Обязателен | Описание |
|----------------------|--------|------------|----------|
| `pg`                 | string | ✅          | `water` |
| `input`              | bool   | ✅          | `true` — входящие, `false` — исходящие |
| `dateFrom`           | string | нет        | UTC ISO 8601 с мс: `2024-01-01T00:00:00.000Z` |
| `dateTo`             | string | нет        | UTC ISO 8601 с мс |
| `limit`              | int    | нет        | Константа **50**, макс. 1000 |
| `did`                | string | нет        | `number` последнего документа предыдущей страницы |
| `orderedColumnValue` | string | нет        | `docDate` последнего документа предыдущей страницы |

> `did` и `orderedColumnValue` — курсорная пагинация. Добавлять в запрос **только если непусты**.

**Defaults при отсутствии параметров фильтрации:**
```go
dateTo   = time.Now().UTC()
dateFrom = time.Now().UTC().Add(-72 * time.Hour) // минус 3 суток
limit    = 50
// did и orderedColumnValue = "" (первая страница)
```

**Go-модели (`internal/model/document.go`):**

```go
type DocListParams struct {
    PG                 string
    Input              bool
    DateFrom           string // UTC ISO 8601 с мс
    DateTo             string
    Limit              int    // константа 50
    DID                string // cursor: number последнего doc
    OrderedColumnValue string // cursor: docDate последнего doc
}

type DocListResponse struct {
    Results  []Document `json:"results"`
    NextPage bool       `json:"nextPage"`
}

type Document struct {
    Number        string `json:"number"`      // системный ID — cursor DID + ссылка /docs/:id
    Type          string `json:"type"`
    DocDate       string `json:"docDate"`     // cursor OrderedColumnValue
    SenderInn     string `json:"senderInn"`
    SenderName    string `json:"senderName"`
    ReceiverInn   string `json:"receiverInn"`
    InvoiceNumber string `json:"invoiceNumber"` // человекочитаемый номер (напр. "УТ-13894")
    Status        string `json:"status"`      // CHECKED_OK | WAIT_ACCEPTANCE | REJECTED | CHECKED_NOT_OK
    Input         bool   `json:"input"`
}
```

---

### GET /api/v4/true-api/doc/{docId}/info — детали документа

> ⚠️ API возвращает **массив** `[]DocInfo`, не объект. Брать `response[0]`; ошибка если массив пуст.

**Query-параметры:**

| Параметр | Тип    | Обязателен | Описание |
|----------|--------|------------|----------|
| `pg`     | string | ✅          | `water` |
| `body`   | bool   | ✅          | `true` — **без этого КИ не возвращаются** |

**Go-модели (`internal/model/document.go`):**

```go
type DocInfo struct {
    Number        string  `json:"number"`
    DocDate       string  `json:"docDate"`
    ReceivedAt    string  `json:"receivedAt"`
    Type          string  `json:"type"`
    Status        string  `json:"status"`
    SenderInn     string  `json:"senderInn"`
    SenderName    string  `json:"senderName"`
    ReceiverInn   string  `json:"receiverInn"`
    ReceiverName  string  `json:"receiverName"`
    InvoiceNumber string  `json:"invoiceNumber"` // заголовок страницы деталей
    InvoiceDate   string  `json:"invoiceDate"`
    RelatedDocID  *string `json:"relatedDocId"`
    TurnoverType  string  `json:"turnoverType"`
    Body          DocBody `json:"body"`
}

type DocBody struct {
    Products  []Product `json:"products"`  // primary для water
    CisesList []string  `json:"cisesList"` // fallback если products пуст
    SumNds    string    `json:"sumNds"`
}

type Product struct {
    Code     string `json:"code"`     // КИ — отображать as-is, моноширинным шрифтом
    Name     string `json:"name"`
    CodeType string `json:"codeType"` // OSU | SSCC | ...
    GTIN     string `json:"gtin"`
    Quantity string `json:"quantity"`
    StrNum   int    `json:"strNum"`
}
```

---

## Handlers

### ListDocuments — GET /docs/incoming, GET /docs/outgoing

```
internal/web/handlers/documents.go → func ListDocuments(c *fiber.Ctx) error
```

1. Направление: path содержит `incoming` → `input=true`, `outgoing` → `input=false`.
2. JWT из cookie `crpt_token`; отсутствует → редирект `/auth`.
3. Query Params: `date_from`, `date_to`, `did`, `ordered_column_value`. Если пусты — применить defaults.
4. Вызвать `crpt.Client.ListDocuments(ctx, token, params)`.
5. Cursor следующей страницы (если `nextPage=true`):
```go
last := results[len(results)-1]
nextDID := last.Number
nextOCV := last.DocDate
```
6. Передать в шаблон: `results`, `nextPage`, текущие `dateFrom`/`dateTo`, `nextDID`, `nextOCV`, флаг `isIncoming`.

### ShowDocument — GET /docs/:id

```
internal/web/handlers/documents.go → func ShowDocument(c *fiber.Ctx) error
```

1. `docId` из `c.Params("id")`.
2. JWT из cookie; отсутствует → редирект `/auth`.
3. Вызвать `crpt.Client.GetDocumentInfo(ctx, token, docId, "water")`.
4. Десериализовать `[]DocInfo`, взять `[0]`.
5. `body.products` (primary) или `body.cisesList` (fallback).
6. Передать в `views/document_detail.templ`.

### crpt.Client — методы (`internal/crpt/documents.go`)

```go
func (c *Client) ListDocuments(ctx context.Context, token string, p model.DocListParams) (*model.DocListResponse, error)
// GET /api/v4/true-api/doc/list
// did и orderedColumnValue добавлять только если p.DID != ""

func (c *Client) GetDocumentInfo(ctx context.Context, token, docID, pg string) (*model.DocInfo, error)
// GET /api/v4/true-api/doc/{docID}/info?pg={pg}&body=true
// Ответ []DocInfo → вернуть [0]; если пусто — error
```

---

## UI — шаблоны (views/)

### documents.templ

- Вкладки «Входящие» / «Исходящие» → `/docs/incoming`, `/docs/outgoing`. Активная вкладка выделена.
- Форма фильтрации: `date_from`, `date_to` (`<input type="date">`), GET на текущий маршрут.
- Таблица: `Дата` (`02.01.2006`) | `Счёт №` (`invoiceNumber`) | `Поставщик` | `ИНН` | `Тип` | `Статус` (badge) | `Детали`.
- Статус-badge: `CHECKED_OK`→зелёный, `WAIT_ACCEPTANCE`→жёлтый, `REJECTED`/`CHECKED_NOT_OK`→красный, остальные→серый.
- Пагинация (если `nextPage=true`):
```
/docs/incoming?date_from={dateFrom}&date_to={dateTo}&did={nextDID}&ordered_column_value={nextOCV}
```

### document_detail.templ

- Заголовок: `invoiceNumber` (не `number`).
- Метаданные (2 колонки): дата, поставщик + ИНН, получатель + ИНН, тип, статус-badge, НДС.
- Таблица КИ: `#` | `Наименование` | `GTIN` | `Кол-во` | `Тип кода` | `Код маркировки` (моноширинный `<code>`).
- Debug-секция `<details><summary>Debug</summary>`: `number`, `receivedAt`, `relatedDocId`, `turnoverType`, полный JSON в `<pre>`.

---

## Обработка ошибок

| HTTP | Действие |
|---|---|
| 401 | Редирект на `/auth` |
| 400 | Логировать `error_message`, вернуть ошибку пользователю |
| 403 | Нет доступа у сертификата — показать сообщение |
| 404 | Залогировать полный URL запроса |
| 5xx | Логировать, показать «сервис временно недоступен» |

## Constraints

- `sign-service` only runs on Windows — gRPC calls will fail locally unless tunnelled or mocked.
- JWT is not cached; auth must be repeated after expiry (~10 h).
- **True API v4 only** — пути без `/api/v4/` prefix возвращают 404.
- `pg=water` обязателен в каждом запросе к doc API.
- `body=true` обязателен для `/doc/{id}/info`.
- Rate limit True API: **50 req/s**.
- Формат дат для API — UTC с миллисекундами. Пример форматирования:
```go
t.UTC().Format("2006-01-02T15:04:05.000Z")
```

## Что НЕ реализовывать на этом этапе

- Приёмка / отклонение документа
- Кэширование токена и автообновление
- Экспорт в Excel/CSV
- Фильтрация по `documentType` через UI
- Фильтрация по статусу через UI
