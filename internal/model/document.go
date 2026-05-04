package model

type Document struct {
	ID               string
	Type             string
	Status           string
	SenderName       string
	SenderINN        string
	ReceiverName     string
	ReceiverINN      string
	DocDate          string
	CreatedTimestamp int64
	TotalItems       int
}
