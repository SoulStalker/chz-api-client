package model

type Document struct {
	MessageId        string
	EntityId         string
	DocumentNumber   string
	DocumentDate     string
	CounteragentId   string
	CounteragentName string
	Status           string
	HasMarkingCodes  bool
}

type DocumentDetail struct {
	Document
	MarkingCodes []string
	XML          string
}

type Organization struct {
	BoxId string
	Name  string
	Inn   string
}
