# edo-client — CLAUDE.md

## Назначение проекта

Go-сервис для интеграции с двумя внешними системами:
- **Честный знак (CRPT GIS MT)** — мониторинг маркированных товаров
- **Диадок (Kontур EDО)** — получение входящих документов ЭДО (УПД, накладные)

Криптографическая подпись делегирована в `sign-service` (gRPC, Windows Certificate Store, ГОСТ).

## Цели первой итерации (MVP)

1. Авторизация в Диадок (логин/пароль → Bearer token)
2. Авторизация в Честный знак (OAuth2 через сертификат sign-service)
3. Выбор организации (boxId в Диадок, participantId в ЧЗ)
4. Список входящих документов Диадок
5. Детализация документа (включая марки ЧЗ из `НомСредИдентТов/КИЗ`)
6. Веб-интерфейс: просмотр сертификатов sign-service, выбор сертификата, список документов, детализация

## Стек

| Слой | Технология |
|---|---|
| HTTP-клиент | `go-resty/resty/v2` |
| Конфиг | `ilyakaznacheev/cleanenv` (YAML + ENV) |
| Веб-сервер | `gofiber/fiber/v2` |
| Шаблоны | `a-h/templ` (генерируется в `gen/templ/`) |
| Логирование | `log/slog` canonical log line (один structured лог на запрос) |
| sign-service | gRPC клиент `github.com/SoulStalker/sign-service/gen/signer` |
| Go | 1.24+ |

## Структура проекта

```
edo-client/
├── cmd/
│   └── server/
│       └── main.go              # Точка входа: config → deps → fiber.App
├── config/
│   ├── config.go                # Структуры конфига (cleanenv tags)
│   └── example.yml              # Пример конфига
├── internal/
│   ├── diadoc/
│   │   ├── client.go            # HTTP-клиент Диадок (go-resty)
│   │   ├── auth.go              # Авторизация (login/password → token)
│   │   ├── organizations.go     # Список ящиков (GetMyOrganizations)
│   │   ├── documents.go         # GetDocuments V3 (входящие)
│   │   └── detail.go            # GetMessage + парсинг КИЗ из XML
│   ├── crpt/
│   │   ├── client.go            # HTTP-клиент CRPT (go-resty)
│   │   └── auth.go              # OAuth2 через sign-service (заглушка на старте)
│   ├── signer/
│   │   └── client.go            # gRPC-клиент sign-service (ListCertificates, Sign)
│   ├── web/
│   │   ├── handlers/
│   │   │   ├── certs.go         # GET /certs — список сертификатов
│   │   │   ├── auth.go          # POST /auth — форма входа Диадок
│   │   │   ├── orgs.go          # GET /orgs — выбор организации
│   │   │   ├── documents.go     # GET /docs — список документов
│   │   │   └── detail.go        # GET /docs/:msgId — детализация + КИЗ
│   │   └── middleware/
│   │       └── logger.go        # Canonical log line middleware (slog)
│   └── model/
│       ├── diadoc.go            # DTO: Document, DocumentDetail, MarkingCode
│       └── signer.go            # DTO: Certificate
├── views/                       # templ-шаблоны (*.templ)
│   ├── layout.templ
│   ├── certs.templ
│   ├── auth.templ
│   ├── orgs.templ
│   ├── documents.templ
│   └── detail.templ
├── gen/
│   └── templ/                   # Авто-генерация (не редактировать вручную)
├── Makefile
└── CLAUDE.md
```

## Конфиг (config/config.go)

```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Diadoc   DiadocConfig   `yaml:"diadoc"`
    CRPT     CRPTConfig     `yaml:"crpt"`
    Signer   SignerConfig   `yaml:"signer"`
    Log      LogConfig      `yaml:"log"`
}

type ServerConfig struct {
    Addr string `yaml:"addr" env:"SERVER_ADDR" env-default:":8080"`
}

type DiadocConfig struct {
    BaseURL   string `yaml:"base_url"   env:"DIADOC_BASE_URL"   env-default:"https://diadoc-api.kontur.ru"`
    ClientID  string `yaml:"client_id"  env:"DIADOC_CLIENT_ID"`   // DDauth ключ интеграции
}

type CRPTConfig struct {
    BaseURL string `yaml:"base_url" env:"CRPT_BASE_URL" env-default:"https://markirovka.crpt.ru"`
}

type SignerConfig struct {
    Addr string `yaml:"addr" env:"SIGNER_ADDR" env-default:"localhost:50051"`
}

type LogConfig struct {
    Level string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
}
```

## Авторизация Диадок

**Метод:** `POST /V3/Authenticate`
**Заголовок:** `Authorization: DiadocAuth ddauth_api_client_id={ClientID},ddauth_login={login},ddauth_password={password_sha256_hex}`

Пароль хешируется SHA-256 на клиенте перед отправкой.
Ответ: plain text Bearer token.

Токен хранится в **сессии Fiber** (cookie, in-memory store, не Redis на старте).
При каждом запросе к Диадок: `Authorization: Bearer {token}`.

## Список документов Диадок (GET /V3/GetDocuments)

Параметры для входящих:
```
filterCategory=UniversalTransferDocument.InboundNotFinished
boxId={выбранный boxId}
```

Ответ — `DocumentList` (JSON). Нужные поля модели:
```go
type Document struct {
    MessageId      string    `json:"MessageId"`
    EntityId       string    `json:"EntityId"`
    DocumentNumber string    `json:"DocumentNumber"`
    DocumentDate   string    `json:"DocumentDate"`
    CounteragentId string    // из Counteragent.OrgId
    Status         string    // из DocflowStatus
    HasMarkingCodes bool     // вычислять при парсинге
}
```

## Детализация документа и КИЗ

**Метод:** `GET /V3/GetMessage?boxId={}&messageId={}&entityId={}`

Ответ содержит сущности (`Entities`), в том числе XML-контент документа (base64).
Для извлечения КИЗ (кодов идентификации) — парсить XML:
```xml
<НомСредИдентТов>
    <КИЗ>010460406000002821xVjKN2L0pqT</КИЗ>
</НомСредИдентТов>
```

Парсинг через `encoding/xml`. Не использовать строгие схемы — документы многоформатные.
КИЗ — строки длиной 31 символ (DataMatrix GS1).

## gRPC sign-service (internal/signer/client.go)

```go
import pb "github.com/SoulStalker/sign-service/gen/signer"

// Список сертификатов для отображения в UI
certs, err := pb.NewSignerClient(conn).ListCertificates(ctx, &pb.Empty{})

// Подпись (для ЧЗ авторизации — на будущее)
resp, err := pb.NewSignerClient(conn).Sign(ctx, &pb.SignRequest{
    Payload:    data,
    Thumbprint: selectedThumbprint,
    CallerId:   "edo-client",
})
```

## Canonical Log Line (middleware/logger.go)

Один `slog` лог на HTTP-запрос, в конце обработчика:
```
level=INFO msg="http request"
  method=GET path=/docs status=200
  duration_ms=42 box_id=xxx
  doc_count=25 trace_id=uuid
```

Реализация через `fiber.Middleware` + `slog.With(...)`.
`trace_id` — `X-Request-ID` из заголовка или генерация UUID.

## Веб-интерфейс (Fiber + templ)

Маршруты:
```
GET  /                  → redirect /certs
GET  /certs             → список сертификатов из sign-service
POST /auth              → форма входа Диадок (login, password, thumbprint)
GET  /orgs              → список организаций (ящиков)
POST /orgs/select       → выбор boxId → сессия
GET  /docs              → список входящих документов
GET  /docs/:messageId   → детализация + КИЗ
```

Минималистичный UI: таблицы, формы без JS-фреймворка.
CSS: один встроенный `<style>` в layout.templ (Tailwind CDN для прототипа).

## Makefile targets

```makefile
.PHONY: templ build run lint

templ:       ## Генерация templ → Go
    templ generate ./views/...

build: templ ## Сборка бинаря
    go build -o ./bin/edo-client ./cmd/server

run: build
    ./bin/edo-client --config config/example.yml

lint:
    golangci-lint run ./...
```

## Ограничения первой итерации

- Авторизация ЧЗ (OAuth2 через сертификат) — **заглушка** (`crpt/auth.go` возвращает `ErrNotImplemented`)
- Нет постраничности документов (берём первые 50)
- Нет кэширования токенов в Redis — только in-memory сессия Fiber
- Windows-only: sign-service (crypt32.dll), само приложение запускается на Windows-хосте

## Зависимости (go.mod стартовые)

```
require (
    github.com/gofiber/fiber/v2          latest
    github.com/go-resty/resty/v2         latest
    github.com/ilyakaznacheev/cleanenv   latest
    github.com/a-h/templ                 latest
    github.com/gofiber/storage/memory    latest  // in-memory сессии
    github.com/SoulStalker/sign-service  v0.0.0  // replace → ../sign-service
    google.golang.org/grpc               latest
    google.golang.org/protobuf           latest
)
```

## Порядок имплементации (для Claude Code)

1. `go mod init` + добавить зависимости
2. `config/config.go` + `config/example.yml`
3. `internal/signer/client.go` — gRPC клиент
4. `internal/diadoc/client.go` + `auth.go`
5. `internal/diadoc/organizations.go` + `documents.go` + `detail.go`
6. `internal/model/` — DTO structs
7. `views/*.templ` — шаблоны
8. `make templ`
9. `internal/web/handlers/` — все хендлеры
10. `internal/web/middleware/logger.go`
11. `cmd/server/main.go` — сборка зависимостей, запуск Fiber
12. `Makefile`
13. `internal/crpt/client.go` + `auth.go` (заглушка)

## Что НЕ делать сейчас

- Не писать отправку/подписание документов
- Не реализовывать авторизацию ЧЗ через сертификат (только заглушка)
- Не добавлять БД / очереди
- Не генерировать TypeScript/JS (только server-side rendering)
