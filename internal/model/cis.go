package model

type CisInfo struct {
	CisKey     string
	GTIN       string
	Name       string
	PackType   string // BOX | GROUP | UNIT (from generalPackageType)
	ChildCount int    // len(ChildCodes)
	ChildCodes []string  // raw child codes as returned by API
	Child      []CisChild // enriched children, populated by handlers
}

type CisChild struct {
	CisKey     string
	GTIN       string
	Name       string
	PackType   string
	ChildCount int
}
