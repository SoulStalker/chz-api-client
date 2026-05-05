package crpt

import (
	"context"
	"errors"

	"github.com/go-resty/resty/v2"
)

var ErrUnauthorized = errors.New("crpt: токен истёк или недействителен")

// Signer — минимальный интерфейс для подписи данных через sign-service.
type Signer interface {
	Sign(ctx context.Context, data []byte, thumbprint, callerID string) ([]byte, error)
}

type Client struct {
	http    *resty.Client
	signer  Signer
	baseURL string
}

func New(baseURL string, signer Signer) *Client {
	r := resty.New().
		SetBaseURL(baseURL).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	return &Client{http: r, signer: signer, baseURL: baseURL}
}
