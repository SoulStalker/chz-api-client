package model

type DocListParams struct {
	PG                 string
	Input              bool
	DateFrom           string
	DateTo             string
	Limit              int
	DID                string
	OrderedColumnValue string
}

type DocListResponse struct {
	Results  []Document `json:"results"`
	NextPage bool       `json:"nextPage"`
}

type Document struct {
	Number        string `json:"number"`
	Type          string `json:"type"`
	DocDate       string `json:"docDate"`
	SenderInn     string `json:"senderInn"`
	SenderName    string `json:"senderName"`
	ReceiverInn   string `json:"receiverInn"`
	InvoiceNumber string `json:"invoiceNumber"`
	Status        string `json:"status"`
	Input         bool   `json:"input"`
}

type DocInfo struct {
	Number        string  `json:"number"`
	DocDate       string  `json:"docDate"`
	ReceivedAt    string  `json:"receivedAt"`
	Type          string  `json:"type"`
	Status        string  `json:"status"`
	SenderInn     string  `json:"senderInn"`
	SenderName    string  `json:"senderName"`
	ReceiverInn   string  `json:"receiverInn"`
	ReceiverName  string  `json:"receiverName"`
	InvoiceNumber string  `json:"invoiceNumber"`
	InvoiceDate   string  `json:"invoiceDate"`
	RelatedDocID  *string `json:"relatedDocId"`
	TurnoverType  string  `json:"turnoverType"`
	Body          DocBody `json:"body"`
}

type DocBody struct {
	UPD            string    `json:"upd"`
	UPDDate        string    `json:"updDate"`
	AcceptanceCode string    `json:"acceptanceCode"`
	SumNds         string    `json:"sumNds"`
	Products       []Product `json:"products"`
	CisesList      []string  `json:"cisesList"`
}

type Product struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	CodeType string `json:"codeType"`
	GTIN     string `json:"gtin"`
	Quantity string `json:"quantity"`
	StrNum   int    `json:"strNum"`
}
