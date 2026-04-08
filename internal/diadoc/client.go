package diadoc

import (
	"github.com/go-resty/resty/v2"
)

type Client struct {
	http     *resty.Client
	clientID string
}

func New(baseURL, clientID string) *Client {
	r := resty.New().
		SetBaseURL(baseURL).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	return &Client{http: r, clientID: clientID}
}

func (c *Client) SetToken(token string) {
	c.http.SetHeader("Authorization", "Bearer "+token)
}
