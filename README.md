# chz-api-client

Go-сервис для интеграции с **Честным знаком (CRPT GIS MT True API)**.

Криптографическая подпись делегирована во внешний сервис `sign-service` (gRPC, Windows Certificate Store, ГОСТ).

## Функциональность

- **Авторизация в Честный знак через УКЭП** (двухшаговый CAdES-BES, JWT до 10 ч)
- Просмотр сертификатов из `sign-service`
- **Получение входящих документов с марками** (тип `LP_ACCEPT_GOODS`) с фильтрацией по периоду и статусу

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
chz-api-client/
├── cmd/
│   ├── server/main.go           # Точка входа веб-сервера
│   └── crpt-explore/main.go     # Утилита для отладки авторизации
├── config/
│   ├── config.go                # Структуры конфига
│   └── example.yml              # Пример конфига
├── internal/
│   ├── crpt/
│   │   ├── client.go            # HTTP-клиент CRPT
│   │   ├── auth.go              # Авторизация (GET /auth/key + POST /auth/simpleSignIn)
│   │   └── documents.go         # Входящие документы (GET /true-api/facade/doc)
│   ├── signer/
│   │   └── client.go            # gRPC-клиент sign-service
│   ├── web/
│   │   ├── handlers/
│   │   │   ├── auth.go          # POST/GET /auth → JWT в cookie
│   │   │   ├── certs.go         # GET /certs → список сертификатов
│   │   │   └── documents.go     # GET /docs → входящие документы
│   │   └── middleware/
│   │       └── logger.go        # Canonical log line (slog)
│   └── model/                   # DTO-структуры
├── views/                       # templ-шаблоны (auth, certs, documents, layout)
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

crpt:
  base_url: "https://markirovka.crpt.ru"
  thumbprint: ""          # SHA1 hex сертификата (CRPT_THUMBPRINT)
  inn: "0001112223"       # ИНН организации (CRPT_INN)
  mchd: ""                # Номер МЧД, если требуется (CRPT_MCHD)

signer:
  addr: "localhost:50051" # адрес sign-service gRPC (SIGNER_ADDR)

log:
  level: "info"           # LOG_LEVEL
```

Все параметры можно переопределить переменными окружения: `SERVER_ADDR`, `CRPT_BASE_URL`, `CRPT_THUMBPRINT`, `CRPT_INN`, `CRPT_MCHD`, `SIGNER_ADDR`, `LOG_LEVEL`.

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
| `GET` | `/auth` | Форма авторизации (thumbprint, ИНН, МЧД) |
| `POST` | `/auth` | Авторизация → JWT сохраняется в cookie `crpt_token` |
| `GET` | `/docs` | Входящие документы с марками (`LP_ACCEPT_GOODS`) |

### Параметры `/docs`

| Query | Описание | Пример |
|---|---|---|
| `date_from` | Начало периода (ISO 8601) | `2024-01-01` |
| `date_to` | Конец периода (ISO 8601) | `2024-12-31` |
| `status` | Фильтр по статусу | `WAIT_ACCEPTANCE` |
| `page_num` | Номер страницы (от 0) | `0` |

## Makefile

```
make templ   # Генерация templ → Go
make build   # Сборка бинаря (включает templ)
make run     # Сборка + запуск
make lint    # golangci-lint
```

## Ограничения

- `sign-service` работает только на Windows (crypt32.dll, ГОСТ)
- JWT Честного знака не кэшируется — при необходимости запрашивается заново
- Пагинация в UI не реализована (отображается первая страница, 50 записей)

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
