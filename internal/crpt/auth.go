package crpt

import "errors"

var ErrNotImplemented = errors.New("crpt: oauth2 via certificate not implemented")

func (c *Client) Authenticate(thumbprint string) (string, error) {
	return "", ErrNotImplemented
}
