CLAUDE.md

# Этап 2

## Цель этапа

Реализовать получение JWT-токена ЧЗ (GIS MT True API) через УКЭП-сертификат из sign-service.
После успешного получения токена — убедиться, что он принимается защищёнными методами API.

## Протокол авторизации (раздел 1.5 документации True API v6.44)

Двухшаговый процесс:

### Шаг 1 — GET /auth/key
Сервер возвращает случайную строку для подписи:
```json
{"uuid": "a63ff582-b723-4da7-958b-453da27a6c62", "data": "GNUFBAZBMPIUUMLXNMIOGSHTGFXZMT"}
```

### Шаг 2 — POST /auth/simpleSignIn
Клиент отправляет uuid и подписанные данные:
```json
{
  "uuid": "a63ff582-b723-4da7-958b-453da27a6c62",
  "data": "<base64 CAdES-BES прикреплённой подписи>"
}
```
Ответ при успехе:
```json
{"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}
```
Токен действует **не более 10 часов**.

### Критичные детали подписи
- Тип: **CAdES-BES, прикреплённая (attached/enveloping)** — `detached = false`
- Формат: CMS/PKCS#7, результат в **Base64**
- Входные данные: строка `data` как есть (не декодировать из base64 перед подписью)
- sign-service уже реализует именно такую подпись через `crypt32.dll`

## Структура проекта (только для этого этапа)

```
crpt-explore/
├── go.mod
├── go.sum
├── main.go          # explore-скрипт: auth + проверочный запрос
└── CLAUDE.md
```

После успешной разведки — перенести в `internal/crpt/` основного сервиса.

## Зависимости

```
github.com/go-resty/resty/v2
github.com/ilyakaznacheev/cleanenv
google.golang.org/grpc
google.golang.org/protobuf
github.com/SoulStalker/sign-service  // replace ../sign-service
```

## Конфигурация (ENV или флаги)

| ENV | Описание | Пример |
|---|---|---|
| `CRPT_BASE_URL` | Боевой или тестовый стенд | `https://markirovka.crpt.ru` |
| `CRPT_THUMBPRINT` | SHA1 hex сертификата из sign-service | `195934d72dcdf...` |
| `SIGNER_ADDR` | gRPC адрес sign-service | `localhost:50051` |

## Алгоритм реализации (main.go)

```
1. Подключиться к sign-service gRPC
2. ListCertificates → показать список → взять thumbprint из ENV/флага
3. GET {CRPT_BASE_URL}/api/v3/auth/key → {uuid, data}
4. Sign(payload=[]byte(data), thumbprint, caller_id="crpt-explore") → signedBytes
5. base64.StdEncoding.EncodeToString(signedBytes) → signedB64
6. POST /api/v3/auth/simpleSignIn body={uuid, data: signedB64} → {token}
7. Напечатать токен и его длину
8. Проверочный запрос с токеном (например GET /api/v3/facade/edo/documents) → статус
```

## URL стендов ЧЗ

- **Продуктив**: `https://markirovka.crpt.ru`
- **Sandbox**: уточнить в ЛК ГИС МТ (обычно `https://markirovka-sandbox.crpt.ru`)

Базовый путь для v3: `/api/v3/`

## Canonical log line

Каждый HTTP-запрос логировать через `slog`:
```
level=INFO msg="crpt request" method=GET url=/auth/key status=200 duration_ms=123
level=INFO msg="sign" thumbprint=195934... payload_len=30 signed_len=2048
level=INFO msg="crpt request" method=POST url=/auth/simpleSignIn status=200
level=INFO msg="token received" token_len=512 expires_in="10h"
```

## Обработка ошибок (коды из документации)

| HTTP | Тело | Действие |
|---|---|---|
| 400 | uuid не указан | проверить шаг 1 |
| 400 | Ошибка при проверке подписи | неверный формат подписи (detached?) |
| 400 | UUID не найден | uuid устарел — повторить шаг 1 |
| 403 | Отсутствует доступ | сертификат не добавлен в ЛК ГИС МТ |

## Что НЕ реализовывать на этом этапе

- Получение документов ЭДО
- Веб-интерфейс
- Кэширование токена
- Retry-логику (просто логировать ошибку и выходить)

## После успешной разведки

Зафиксировать:
1. Точный URL стенда
2. Реальный формат ответа `/auth/key` (поля могут отличаться от документации)
3. Длину и структуру JWT (для последующего парсинга `exp`)
4. Какие защищённые методы доступны с полученным токеном

---

# Этап 3

## Цель этапа

Реализовать получение списка входящих документов с марками (УПД/передаточные документы), которые поступают в организацию через ГИС МТ.
Результат — метод `Client.IncomingDocuments(ctx, filter)` в `internal/crpt/` и отображение списка в веб-интерфейсе.

## Эндпоинт API (True API v3, раздел 4.1 документации)

```
GET /api/v3/true-api/facade/doc
Authorization: Bearer {JWT}
```

### Query-параметры

| Параметр | Тип | Описание |
|---|---|---|
| `dateFrom` | string (ISO 8601) | Начало периода, напр. `2024-01-01` |
| `dateTo` | string (ISO 8601) | Конец периода |
| `type` | string | Тип документа (см. ниже) |
| `status` | string | Статус документа (см. ниже) |
| `pageNum` | int | Номер страницы, начиная с 0 |
| `pageSize` | int | Размер страницы, макс. 10000 |

### Релевантные типы документов для входящих товаров

| Тип | Описание |
|---|---|
| `LP_ACCEPT_GOODS` | Приёмка маркированных товаров (организация — получатель) |
| `LP_SHIP_GOODS` | Отгрузка (для исходящих, не нужен на этом этапе) |

Фильтрация по организации: сервер автоматически отдаёт документы текущего участника (по JWT).
Для входящих — документы, где `receiverInn` совпадает с ИНН из сертификата.

### Статусы документов

| Статус | Описание |
|---|---|
| `WAIT_ACCEPTANCE` | Ожидает приёмки получателем |
| `ACCEPTED` | Принято |
| `REJECTED` | Отклонено |
| `CANCELLED` | Отменено |

### Пример ответа

```json
{
  "results": [
    {
      "id": "ba7e7bc8-1234-5678-abcd-000000000001",
      "type": "LP_ACCEPT_GOODS",
      "status": "WAIT_ACCEPTANCE",
      "senderName": "ООО Поставщик",
      "senderInn": "7700000001",
      "receiverName": "ООО Получатель",
      "receiverInn": "7700000002",
      "docDate": "2024-05-01",
      "createdTimestamp": 1714521600000,
      "totalItems": 12
    }
  ],
  "total": 1
}
```

## Структура проекта (изменения)

```
internal/crpt/
├── client.go          # уже существует
├── auth.go            # уже существует
└── documents.go       # NEW: IncomingDocuments, модели Document/DocFilter/DocPage
internal/model/
└── document.go        # NEW: публичная модель документа для веб-слоя
internal/web/handlers/
└── documents.go       # NEW: HTTP-хендлер списка входящих документов
views/
└── documents.templ    # NEW: страница со списком документов
```

## Модели данных

```go
// internal/crpt/documents.go

type DocFilter struct {
    DateFrom string // "2006-01-02", пусто = без ограничения
    DateTo   string
    Type     string // "" = все типы; "LP_ACCEPT_GOODS" для входящих
    Status   string // "" = все статусы
    PageNum  int
    PageSize int    // default 50
}

type Document struct {
    ID               string
    Type             string
    Status           string
    SenderName       string
    SenderINN        string
    ReceiverName     string
    ReceiverINN      string
    DocDate          string
    CreatedTimestamp int64
    TotalItems       int
}

type DocPage struct {
    Results []Document
    Total   int
}
```

## Алгоритм реализации (documents.go)

```
1. Собрать query-параметры из DocFilter (пропускать пустые значения)
2. GET /api/v3/true-api/facade/doc с Bearer-токеном в заголовке Authorization
3. Десериализовать ответ в DocPage
4. Логировать запрос (slog, аналогично auth.go)
5. Вернуть DocPage или ошибку
```

Подпись метода:
```go
func (c *Client) IncomingDocuments(ctx context.Context, token string, f DocFilter) (*DocPage, error)
```

Токен передаётся явно (не хранить в Client) — на этом этапе кэширование не нужно.

## Веб-интерфейс

- Маршрут: `GET /documents`
- Хендлер читает токен из сессии (cookie `crpt_token`, установленной на этапе 2)
- Если токен отсутствует — редирект на `/`
- Параметры фильтра принимаются из query string (`date_from`, `date_to`, `status`)
- По умолчанию: тип `LP_ACCEPT_GOODS`, pageSize=50, pageNum=0
- Страница отображает таблицу: дата, тип, статус, отправитель, кол-во марок

## Canonical log line

```
level=INFO msg="crpt request" method=GET url=/facade/doc status=200 duration_ms=45 total=12
```

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
