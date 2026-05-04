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
