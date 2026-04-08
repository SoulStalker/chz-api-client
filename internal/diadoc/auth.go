package diadoc

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

func (c *Client) Authenticate(login, password string) (string, error) {
	passwordHash := sha256Hex(password)
	authHeader := fmt.Sprintf(
		"DiadocAuth ddauth_api_client_id=%s,ddauth_login=%s,ddauth_password=%s",
		c.clientID, login, passwordHash,
	)

	resp, err := c.http.R().
		SetHeader("Authorization", authHeader).
		Post("/V3/Authenticate")
	if err != nil {
		return "", fmt.Errorf("diadoc authenticate: %w", err)
	}
	if resp.IsError() {
		return "", fmt.Errorf("diadoc authenticate: status %d: %s", resp.StatusCode(), resp.String())
	}

	token := strings.TrimSpace(resp.String())
	if token == "" {
		return "", errors.New("diadoc authenticate: empty token")
	}
	return token, nil
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
