package model

type CisInfo struct {
	CisKey     string     `json:"cisKey"`
	GTIN       string     `json:"gtin"`
	Name       string     `json:"name"`
	PackType   string     `json:"packType"`
	ChildCount int        `json:"childCount"`
	Child      []CisChild `json:"child"`
}

type CisChild struct {
	CisKey   string `json:"cisKey"`
	GTIN     string `json:"gtin"`
	Name     string `json:"name"`
	PackType string `json:"packType"`
}
