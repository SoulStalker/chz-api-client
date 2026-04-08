package model

type Certificate struct {
	Thumbprint string
	Subject    string
	Issuer     string
	NotAfter   string
}
