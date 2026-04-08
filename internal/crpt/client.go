package crpt

import (
	"context"

	"github.com/go-resty/resty/v2"
)

// Signer — минимальный интерфейс для подписи данных через sign-service.
type Signer interface {
	Sign(ctx context.Context, data []byte, thumbprint string) ([]byte, error)
}

type Client struct {
	http   *resty.Client
	signer Signer
}

func New(baseURL string, signer Signer) *Client {
	r := resty.New().
		SetBaseURL(baseURL).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	return &Client{http: r, signer: signer}
}
