# edo-client

Go-сервис для интеграции с системами **Диадок (Контур ЭДО)** и **Честный знак (CRPT GIS MT)**.

Позволяет просматривать входящие документы ЭДО (УПД, накладные) и извлекать коды маркировки (КИЗ) из XML-содержимого документов через минималистичный веб-интерфейс.

Криптографическая подпись делегирована во внешний сервис `sign-service` (gRPC, Windows Certificate Store, ГОСТ).

## Функциональность

- Авторизация в Диадок (логин/пароль → Bearer token)
- **Авторизация в Честный знак через УКЭП** (двухшаговый CAdES-BES, JWT до 10 ч)
- Просмотр сертификатов из `sign-service`
- Выбор организации (boxId)
- Список входящих документов (УПД)
- Детализация документа с извлечением КИЗ из XML (`НомСредИдентТов/КИЗ`)

## Стек

| Слой | Технология |
|---|---|
| HTTP-клиент | `go-resty/resty/v2` |
| Конфиг | `ilyakaznacheev/cleanenv` (YAML + ENV) |
| Веб-сервер | `gofiber/fiber/v2` |
| Шаблоны | `a-h/templ` |
| Логирование | `log/slog` (canonical log line) |
| sign-service | gRPC (`github.com/SoulStalker/sign-service`) |
| Go | 1.24+ |

## Структура проекта

```
edo-client/
├── cmd/server/main.go           # Точка входа
├── config/
│   ├── config.go                # Структуры конфига
│   └── example.yml              # Пример конфига
├── internal/
│   ├── diadoc/                  # HTTP-клиент Диадок (авторизация, документы, детализация)
│   ├── crpt/                    # HTTP-клиент CRPT (авторизация через УКЭП, JWT)
│   ├── signer/                  # gRPC-клиент sign-service
│   ├── web/
│   │   ├── handlers/            # Fiber-хендлеры
│   │   └── middleware/          # Canonical log line (slog)
│   └── model/                   # DTO-структуры
├── views/                       # templ-шаблоны
├── Makefile
└── CLAUDE.md
```

## Требования

- Go 1.24+
- [`templ`](https://templ.guide/) CLI (`go install github.com/a-h/templ/cmd/templ@latest`)
- `sign-service` репозиторий рядом (`../sign-service`) с запущенным gRPC-сервером на Windows

## Конфигурация

Скопируйте пример и заполните нужные значения:

```bash
cp config/example.yml config/config.yml
```

```yaml
server:
  addr: ":8080"

diadoc:
  base_url: "https://diadoc-api.kontur.ru"
  client_id: "YOUR_DIADOC_CLIENT_ID"   # ключ интеграции DDauth

crpt:
  base_url: "https://markirovka.crpt.ru"
  thumbprint: "195934d72dcdf..."   # SHA1 hex сертификата (CRPT_THUMBPRINT)

signer:
  addr: "localhost:50051"   # адрес sign-service gRPC

log:
  level: "info"
```

Все параметры можно переопределить переменными окружения: `SERVER_ADDR`, `DIADOC_CLIENT_ID`, `CRPT_BASE_URL`, `CRPT_THUMBPRINT`, `SIGNER_ADDR`, `LOG_LEVEL` и др.

## Запуск

```bash
# Генерация шаблонов + сборка + запуск
make run
```

Или по шагам:

```bash
make templ   # генерация templ → Go
make build   # сборка бинаря в ./bin/edo-client
./bin/edo-client --config config/config.yml
```

## Маршруты

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/` | Редирект на `/certs` |
| `GET` | `/certs` | Список сертификатов из sign-service |
| `POST` | `/auth` | Вход в Диадок (логин, пароль, отпечаток сертификата) |
| `GET` | `/orgs` | Список организаций (ящиков) |
| `POST` | `/orgs/select` | Выбор boxId → сохраняется в сессии |
| `GET` | `/docs` | Список входящих документов (УПД) |
| `GET` | `/docs/:messageId` | Детализация документа + список КИЗ |

## Makefile

```
make templ   # Генерация templ → Go
make build   # Сборка бинаря (включает templ)
make run     # Сборка + запуск
make lint    # golangci-lint
```

## Ограничения MVP

- Нет постраничности — загружается до 50 документов
- Сессии хранятся in-memory (перезапуск сервера сбрасывает авторизацию)
- `sign-service` работает только на Windows (crypt32.dll, ГОСТ)
- JWT Честного знака не кэшируется — при необходимости запрашивается заново
