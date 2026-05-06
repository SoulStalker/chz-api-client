package model

type DocListResponse struct {
	Results  []Document `json:"results"`
	NextPage bool       `json:"nextPage"`
}

type Document struct {
	Number     string `json:"number"`
	Type       string `json:"type"`
	DocDate    string `json:"docDate"`
	SenderInn  string `json:"senderInn"`
	SenderName string `json:"senderName"`
	Status     string `json:"status"`
}

type DocInfoResponse struct {
	Number        string  `json:"number"`
	Type          string  `json:"type"`
	DocDate       string  `json:"docDate"`
	SenderInn     string  `json:"senderInn"`
	SenderName    string  `json:"senderName"`
	ReceiverInn   string  `json:"receiverInn"`
	ReceiverName  string  `json:"receiverName"`
	InvoiceNumber string  `json:"invoiceNumber"`
	Status        string  `json:"status"`
	Body          DocBody `json:"body"`
}

type DocBody struct {
	Products  []Product `json:"products"`
	CisesList []string  `json:"cisesList"`
}

type Product struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	CodeType string `json:"codeType"`
	GTIN     string `json:"gtin"`
	Quantity string `json:"quantity"`
}
