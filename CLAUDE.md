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

- **`internal/crpt/`** — HTTP client wrapping the CRPT True API (`go-resty`). `auth.go`: двухшаговая авторизация GET `/auth/key` → подпись → POST `/auth/simpleSignIn` → JWT в cookie `crpt_token`. `documents.go`: работа со списком и деталями документов (v4). `cises.go`: работа с упаковками КИ (v3).
- **`internal/signer/`** — gRPC client for `sign-service`. Filters certificates from the Windows store, removing non-personal entries (Adobe CAs, UUID-named system certs). The `Signer` interface in `internal/crpt/client.go` is the seam between these two packages.
- **`internal/web/handlers/`** — Fiber HTTP handlers. Auth credentials (thumbprint, INN, МЧД) come from config defaults but can be overridden per-request via form fields.
- **`views/`** — `templ` templates. Every `*.templ` file has a corresponding generated `*_templ.go` that must be regenerated after edits (`make templ`). Never edit `*_templ.go` directly.
- **`config/`** — `cleanenv` reads YAML with ENV overrides. `MustLoad` panics on missing file. Copy `config/example.yml` → `config/config.yml` or `config/prod.yml` to get started.

### Request flow

```
Browser → Fiber (middleware/logger.go) → handler → crpt.Client (v4) → CRPT True API v4
                                                  → crpt.Client (v3) → CRPT True API v3 (cises/info)
                                                  ↘ signer.Client   → sign-service gRPC (Windows)
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
| GET | `/docs/:id` | `handlers.ShowDocument` | Детали УПД + таблица кодов (уровень 1) |
| GET | `/docs/:id/pack/:code` | `handlers.ShowPack` | Транспортная упаковка КИТУ (уровень 2) |
| GET | `/docs/:id/pack/:code/group/:childCode` | `handlers.ShowGroup` | Групповая упаковка КИГУ (уровень 3) |

### Config env overrides

`SERVER_ADDR`, `CRPT_BASE_URL`, `CRPT_THUMBPRINT`, `CRPT_INN`, `CRPT_MCHD`, `SIGNER_ADDR`, `LOG_LEVEL`

---

## API: Documents — True API v4

> ⚠️ Документы: **только v4** (`/api/v4/true-api/...`). Пути v3 для документов возвращают 404.
> ⚠️ Упаковки (cises): **только v3** (`/api/v3/true-api/cises/...`). Разные версии для разных методов.

Base: `{CRPT_BASE_URL}/api/v4/true-api/...`
Auth header: `Authorization: Bearer <JWT>` (из cookie `crpt_token`).
Rate limit: **50 req/s**.

---

### GET /api/v4/true-api/doc/list

**Query-параметры:**

| Параметр             | Тип    | Обязателен | Описание |
|----------------------|--------|------------|----------|
| `pg`                 | string | ✅          | `water` |
| `input`              | bool   | ✅          | `true`=входящие, `false`=исходящие |
| `dateFrom`           | string | нет        | UTC ISO 8601 с мс: `2024-01-01T00:00:00.000Z` |
| `dateTo`             | string | нет        | UTC ISO 8601 с мс |
| `limit`              | int    | нет        | Константа **50** |
| `did`                | string | нет        | Cursor: `number` последнего doc предыдущей страницы |
| `orderedColumnValue` | string | нет        | Cursor: `docDate` последнего doc предыдущей страницы |

> `did` и `orderedColumnValue` добавлять в запрос **только если непусты**.

**Defaults:** `dateTo=now`, `dateFrom=now-72h`, `limit=50`, cursor пуст.

**Go-модели (`internal/model/document.go`):**

```go
type DocListParams struct {
    PG                 string
    Input              bool
    DateFrom           string // UTC ISO 8601 с мс
    DateTo             string
    Limit              int    // константа 50
    DID                string // cursor
    OrderedColumnValue string // cursor
}

type DocListResponse struct {
    Results  []Document `json:"results"`
    NextPage bool       `json:"nextPage"`
}

type Document struct {
    Number        string `json:"number"`        // системный ID — cursor DID + ссылка /docs/:id
    Type          string `json:"type"`
    DocDate       string `json:"docDate"`        // cursor OrderedColumnValue
    SenderInn     string `json:"senderInn"`
    SenderName    string `json:"senderName"`
    ReceiverInn   string `json:"receiverInn"`
    InvoiceNumber string `json:"invoiceNumber"` // человекочитаемый номер (напр. "УТ-13894")
    Status        string `json:"status"`        // CHECKED_OK | WAIT_ACCEPTANCE | REJECTED | CHECKED_NOT_OK
    Input         bool   `json:"input"`
}
```

---

### GET /api/v4/true-api/doc/{docId}/info

> ⚠️ Возвращает **массив** `[]DocInfo`. Брать `[0]`; ошибка если пуст.

**Query:** `pg=water&body=true` (оба обязательны; без `body=true` КИ не возвращаются).

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
    InvoiceNumber string  `json:"invoiceNumber"` // заголовок страницы
    InvoiceDate   string  `json:"invoiceDate"`
    RelatedDocID  *string `json:"relatedDocId"`
    TurnoverType  string  `json:"turnoverType"`
    Body          DocBody `json:"body"`
}

type DocBody struct {
    UPD            string    `json:"upd"`            // номер УПД
    UPDDate        string    `json:"updDate"`
    AcceptanceCode string    `json:"acceptanceCode"`
    SumNds         string    `json:"sumNds"`
    Products       []Product `json:"products"`        // primary для water
    CisesList      []string  `json:"cisesList"`       // fallback если products пуст
}

type Product struct {
    Code     string `json:"code"`     // КИ (SGTIN/КИТУ/КИГУ) — нормализованный, as-is
    Name     string `json:"name"`
    CodeType string `json:"codeType"` // UNIT | GROUP | BOX — см. справочник ниже
    GTIN     string `json:"gtin"`
    Quantity string `json:"quantity"`
    StrNum   int    `json:"strNum"`
}
```

**Справочник типов упаковки:**

| CodeType | Русское название | Уровень |
|----------|-----------------|---------|
| `UNIT`   | Единица товара (SGTIN) | 3 (конечный) |
| `GROUP`  | Групповая упаковка (КИГУ) | 2 |
| `BOX`    | Транспортная упаковка (КИТУ) | 1 |
| `OSU`    | Осн. единица (water) | конечный |

---

## API: Упаковки — True API v3

> Только для получения состава упаковок (КИТУ/КИГУ). Версия v3, не v4.

### POST /api/v3/true-api/cises/info

**Тело запроса:**
```json
["<код_упаковки_нормализованный>"]
```

**Пример:**
```bash
curl -X POST "{CRPT_BASE_URL}/api/v3/true-api/cises/info" \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '["02046070354208943712"]'
```

**Go-модели (`internal/model/cis.go`):**

```go
// Запрос — массив нормализованных кодов
type CisInfoRequest []string

// Ответ — массив объектов, один на каждый переданный код
type CisInfo struct {
    CisKey      string     `json:"cisKey"`      // нормализованный код
    GTIN        string     `json:"gtin"`
    Name        string     `json:"name"`
    PackType    string     `json:"packType"`    // UNIT | GROUP | BOX
    ChildCount  int        `json:"childCount"`  // кол-во вложений
    Child       []CisChild `json:"child"`       // вложенные КИ
}

type CisChild struct {
    CisKey   string `json:"cisKey"`
    GTIN     string `json:"gtin"`
    Name     string `json:"name"`
    PackType string `json:"packType"` // UNIT | GROUP | BOX
}
```

> Коды передавать в **нормализованном виде** (без скобок AI `01` и `21`). Не трансформировать — использовать `code` as-is из ответа doc/info.

**crpt.Client метод (`internal/crpt/cises.go`):**

```go
func (c *Client) GetCisInfo(ctx context.Context, token string, codes []string) ([]model.CisInfo, error)
// POST /api/v3/true-api/cises/info
// Тело: JSON-массив codes
```

---

## Handlers

### ListDocuments — GET /docs/incoming, GET /docs/outgoing

```
internal/web/handlers/documents.go → func ListDocuments(c *fiber.Ctx) error
```

1. Направление: `incoming`→`input=true`, `outgoing`→`input=false`.
2. JWT из cookie; отсутствует → редирект `/auth`.
3. Query: `date_from`, `date_to`, `did`, `ordered_column_value`. Если пусты — defaults.
4. `crpt.Client.ListDocuments(...)`.
5. Cursor (если `nextPage=true`): `last.Number` → `nextDID`, `last.DocDate` → `nextOCV`.
6. Шаблон: `views/documents.templ`.

### ShowDocument — GET /docs/:id (Уровень 1: УПД)

```
internal/web/handlers/documents.go → func ShowDocument(c *fiber.Ctx) error
```

1. `docId` из `c.Params("id")`.
2. JWT из cookie; отсутствует → редирект `/auth`.
3. `crpt.Client.GetDocumentInfo(...)` → `[]DocInfo[0]`.
4. `body.products` (primary) или `body.cisesList` (fallback).
5. Для каждого продукта: если `codeType` ∈ {`BOX`, `GROUP`} → ссылка «Детали» на `/docs/{id}/pack/{code}`.
6. Шаблон: `views/document_detail.templ`.

### ShowPack — GET /docs/:id/pack/:code (Уровень 2: КИТУ)

```
internal/web/handlers/packs.go → func ShowPack(c *fiber.Ctx) error
```

1. `docId` из `c.Params("id")`, `packCode` из `c.Params("code")`.
2. JWT из cookie; отсутствует → редирект `/auth`.
3. `crpt.Client.GetCisInfo(ctx, token, []string{packCode})` → `[0]`.
4. Итерировать `CisInfo.Child`:
   - если `child.PackType == "GROUP"` → ссылка «[N] вложений» на `/docs/{id}/pack/{packCode}/group/{child.CisKey}`.
   - если `child.PackType == "UNIT"` → конечный уровень, ссылка не нужна.
5. Хлебные крошки: `Документы › УПД {invoiceNumber} › Транспортная упаковка {packCode}`.
6. Шаблон: `views/pack_detail.templ`.

### ShowGroup — GET /docs/:id/pack/:code/group/:childCode (Уровень 3: КИГУ)

```
internal/web/handlers/packs.go → func ShowGroup(c *fiber.Ctx) error
```

1. `docId`, `packCode`, `groupCode` из Params.
2. JWT из cookie; отсутствует → редирект `/auth`.
3. `crpt.Client.GetCisInfo(ctx, token, []string{groupCode})` → `[0]`.
4. Отображать `Child` — список UNIT (конечный уровень), колонка «Содержит» пуста.
5. Хлебные крошки: `Документы › УПД {invoiceNumber} › Транспортная упаковка {packCode} › Групповая упаковка {groupCode}`.
6. Шаблон: `views/group_detail.templ`.

---

## UI — шаблоны (views/)

### documents.templ

- Вкладки «Входящие» / «Исходящие» → `/docs/incoming`, `/docs/outgoing`. Активная выделена.
- Форма: `date_from`, `date_to` (`<input type="date">`), GET на текущий маршрут.
- Таблица: `Дата` (`02.01.2006`) | `Счёт №` | `Поставщик` | `ИНН` | `Тип` | `Статус` (badge) | `Детали`.
- Статус-badge: `CHECKED_OK`→зелёный, `WAIT_ACCEPTANCE`→жёлтый, `REJECTED`/`CHECKED_NOT_OK`→красный, остальные→серый.
- Пагинация: кнопка «Следующая страница» если `nextPage=true`, URL: `/docs/incoming?date_from=...&date_to=...&did={nextDID}&ordered_column_value={nextOCV}`.

### document_detail.templ (Уровень 1)

- Заголовок: `invoiceNumber`. Статус-badge.
- Вкладки: «Коды» (по умолчанию) | «Расширенная информация».
- Вкладка «Коды» — таблица: `Код` (моноширинный) | `GTIN` | `Наименование` | `Тип упаковки` | `Содержит`.
  - Колонка «Содержит»: если `codeType` ∈ {`BOX`, `GROUP`} → ссылка `[N] вложений` → `/docs/{id}/pack/{code}`. Если `UNIT`/`OSU` — пусто.
  - N берётся из `quantity` продукта или из ответа cises/info (не делать дополнительный запрос на уровне 1).
- Вкладка «Расширенная информация»: `upd`, `updDate`, `acceptanceCode`, `sumNds`, `turnoverType`, `senderInfo`, `buyerInfo`.
- Debug-секция `<details>`: `number`, `receivedAt`, `relatedDocId`, полный JSON.

### pack_detail.templ (Уровень 2: КИТУ)

- Заголовок: «Транспортная упаковка», подзаголовок: полный код `packCode` моноширинным.
- Хлебные крошки.
- Таблица вложений: `Код вложения` (моноширинный + `name` под ним) | `Тип упаковки` | `GTIN` | `Содержит`.
  - `GROUP` → ссылка `[N] вложений` → `/docs/{id}/pack/{code}/group/{child.CisKey}`.
  - `UNIT` → «—».

### group_detail.templ (Уровень 3: КИГУ)

- Заголовок: «Групповая упаковка», подзаголовок: полный код `groupCode` моноширинным.
- Хлебные крошки.
- Таблица: `Код` (моноширинный) | `GTIN` | `Наименование` | `Тип` | `Содержит` (всегда пусто — конечный уровень).

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
- **doc/list, doc/info — True API v4** (`/api/v4/true-api/...`).
- **cises/info — True API v3** (`/api/v3/true-api/cises/...`). Не перепутать версии.
- `pg=water` обязателен для всех v4-запросов.
- `body=true` обязателен для `/doc/{id}/info`.
- Коды КИ передавать **нормализованными** (без скобок). Не трансформировать — использовать as-is из API.
- Rate limit: **50 req/s**.
- Формат дат для API: `t.UTC().Format("2006-01-02T15:04:05.000Z")`.

## Что НЕ реализовывать на этом этапе

- Приёмка / отклонение документа
- Кэширование токена и автообновление
- Экспорт в Excel/CSV
- Фильтрация по `documentType` через UI
- Фильтрация по статусу через UI

# Problems 

Неверные типы документов для входящих: В README и логике маршрута /docs всё ещё указан тип LP_ACCEPT_GOODS
. Согласно справочнику «Типы документов», LP_ACCEPT_GOODS — это документ приёмки, который создает сам участник
. Чтобы увидеть документы, которые прислали поставщики (входящие УПД), нужно искать типы отгрузок: LP_SHIP_GOODS, LP_SHIP_GOODS_CSV или LP_SHIP_GOODS_XML
.
Устаревшая модель пагинации: В параметрах /docs используется page_num
. True API v4 для метода doc/list использует курсорную пагинацию: поле nextPage (bool) в ответе указывает на наличие данных, а для перехода используются параметры did и orderedColumnValue
. Использование номеров страниц может привести к дублированию или пропуску документов.
Отсутствие кэширования JWT: В ограничениях указано, что токен не кэшируется
. Документация ГИС МТ прямо заявляет, что токен действителен в течение 10 часов
. Постоянные запросы на авторизацию через gRPC sign-service при каждом обновлении списка документов — это лишняя нагрузка, которая может привести к блокировке за превышение лимита в 50 запросов в секунду
.
Риск неполного отображения вложений: Метод doc/{docId}/info по умолчанию возвращает информацию только о 50 дочерних КМ
. Если транспортная упаковка (КИТУ) в УПД содержит 144 вложения (как на вашем макете «транспортная_упаковка.png»), то через детали документа вы увидите только треть из них.

--------------------------------------------------------------------------------
2. Как нужно переделать (Технический план)
А. Исправление логики списка документов
Типы: Для маршрута /docs/incoming использовать фильтр input=true и типы LP_SHIP_GOODS...
.
Пагинация:
В Go-клиенте реализовать поддержку параметров did и orderedColumnValue.
В UI (Templ) кнопку «Далее» формировать, беря number и docDate из последнего элемента текущего списка results
.
Б. Реализация иерархии (Drill-down)
Согласно вашим макетам, логику отображения упаковок нужно разделить:
Уровень 1 (Детали УПД): Вызов GET /api/v4/true-api/doc/{docId}/info?body=true
.
Важно: Если codeType == "OSU", отображать только GTIN и количество (как в строке 1368шт), не делая код ссылкой [История диалога].
Уровень 2 и 3 (КИТУ/КИГУ): Для получения состава упаковки (как в транспортная_упаковка.png) нельзя полагаться на тело документа.
Нужно использовать метод POST /api/v3/true-api/cises/info
.
В теле запроса передавать код упаковки (КИТУ/КИГУ).
В ответе парсить массив child
. Это единственный надежный способ получить полный список вложений (например, все 144 единицы).
В. Оптимизация авторизации
Реализовать прослойку (middleware) или логику в сервисе, которая сохраняет полученный JWT в памяти (или Redis/Cookie с Max-Age) на 9-10 часов.
Перед вызовом sign-service проверять, не истек ли текущий токен.
Резюме по «Добавлены упаковки»
Если в последнем коммите вы реализовали вложения КИТУ/КИГУ только на основе данных из doc/info, это будет работать некорректно для больших упаковок. Переделайте получение состава упаковок на метод /cises/info, передавая туда конкретный КИ, по которому кликнул пользователь

## 
При открытии деталей я вижу такой вид:
20 позиций
Код	GTIN	Наименование	Тип упаковки	Содержит
00146070170904051867		Вода минеральная природная лечебно-столовая питьевая " Аш-тау, скважина 6-К" 0,5 л.с/б. элита /720/	Единица товара	—
00346070170903552345		Вода минеральная природная лечебно-столовая питьевая газированная "Аш-тау, скважина 6-К" 1.5 л. ПЭТ	Единица товара	—

а должно быть:
Код (всего: 20) GTIN Наименование Тип упаковки Содержит
00146070170904051867 - Вода минеральная природная лечебно-столовая питьевая " Аш-тау, скважина 6-К" 0,5 л.с/б. элита /720/ Транспортная упаковка 60 вложений
00346070170903552345 - Вода минеральная природная лечебно-столовая питьевая газированная "Аш-тау, скважина 6-К" 1.5 л. ПЭТ Транспортная упаковка 68 вложений
