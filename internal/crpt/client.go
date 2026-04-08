package crpt

import "github.com/go-resty/resty/v2"

type Client struct {
	http *resty.Client
}

func New(baseURL string) *Client {
	r := resty.New().
		SetBaseURL(baseURL).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	return &Client{http: r}
}
