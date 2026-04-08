module github.com/SoulStalker/chz-api-client

go 1.24

require (
	github.com/gofiber/fiber/v2 v2.52.6
	github.com/go-resty/resty/v2 v2.16.5
	github.com/ilyakaznacheev/cleanenv v1.5.0
	github.com/a-h/templ v0.3.865
	github.com/gofiber/storage/memory/v2 v2.0.0
	github.com/SoulStalker/sign-service v0.0.0
	google.golang.org/grpc v1.73.0
	google.golang.org/protobuf v1.36.6
)

replace github.com/SoulStalker/sign-service => ../sign-service
