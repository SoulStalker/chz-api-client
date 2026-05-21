package model

import "time"

type Zakaz struct {
	Kod          string
	Data         time.Time
	Nomer        string
	PostKod      string
	PostImya     string
	PostINN      string
	DataPostavki time.Time
	NakladKod    string
	NakladNomer  string
	Summa        float64
	UpdatedAt    time.Time
	CRPTDocID    *string
}

type ZakazLine struct {
	ID       int
	ZakazKod string
	TovarKod string
	SHK      string
	Naim     string
	Kolvo    float64
	Summa    float64
}
