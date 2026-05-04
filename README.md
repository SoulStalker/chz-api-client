# chz-api-client

Go-сервис для интеграции с **Честным знаком (CRPT GIS MT True API)**.

Криптографическая подпись делегирована во внешний сервис `sign-service` (gRPC, Windows Certificate Store, ГОСТ).

## Функциональность

- **Авторизация в Честный знак через УКЭП** (двухшаговый CAdES-BES, JWT до 10 ч)
- Просмотр сертификатов из `sign-service`

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
├── cmd/server/main.go           # Точка входа
├── config/
│   ├── config.go                # Структуры конфига
│   └── example.yml              # Пример конфига
├── internal/
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

crpt:
  base_url: "https://markirovka.crpt.ru"
  thumbprint: "195934d72dcdf..."   # SHA1 hex сертификата (CRPT_THUMBPRINT)
  inn: "0001112223"                # ИНН организации (CRPT_INN)

signer:
  addr: "localhost:50051"   # адрес sign-service gRPC

log:
  level: "info"
```

Все параметры можно переопределить переменными окружения: `SERVER_ADDR`, `CRPT_BASE_URL`, `CRPT_THUMBPRINT`, `CRPT_INN`, `SIGNER_ADDR`, `LOG_LEVEL`.

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
