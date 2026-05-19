package model

import "encoding/xml"

type ExportRoot struct {
	XMLName  xml.Name       `xml:"ПриёмкаМаркировки"`
	Version  string         `xml:"ВерсияФормата,attr"`
	Document ExportDocument `xml:"Документ"`
}

type ExportDocument struct {
	UPDNumber    string       `xml:"НомерУПД,attr"`
	UPDDate      string       `xml:"ДатаУПД,attr"`
	SenderINN    string       `xml:"ПоставщикИНН,attr"`
	SenderName   string       `xml:"ПоставщикНаименование,attr"`
	ReceiverINN  string       `xml:"ПолучательИНН,attr"`
	ReceiverName string       `xml:"ПолучательНаименование,attr"`
	Status       string       `xml:"Статус,attr"`
	ZakazKod     string       `xml:"КодЗаказа,attr,omitempty"`
	Codes        []ExportCode `xml:"КМ"`
}

type ExportCode struct {
	Code       string `xml:"Код,attr"`
	Type       string `xml:"Тип,attr"`
	Parent     string `xml:"Родитель,attr"`
	ChildCount int    `xml:"Вложений,attr,omitempty"`
}
