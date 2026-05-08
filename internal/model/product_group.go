package model

// ProductGroup represents a CRPT True API product group (pg parameter).
// Codes must match the values accepted by the CRPT v4 API.
type ProductGroup struct {
	Code  string
	Label string
}

var ProductGroups = []ProductGroup{
	{"water", "Вода"},
	{"milk", "Молочная продукция"},
	{"softdrinks", "Соковая продукция и безалкогольные напитки"},
	{"conserve", "Консервированная продукция"},
	{"petfood", "Корма для животных"},
}

func IsValidPG(pg string) bool {
	for _, g := range ProductGroups {
		if g.Code == pg {
			return true
		}
	}
	return false
}
