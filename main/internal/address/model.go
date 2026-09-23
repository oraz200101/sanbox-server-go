package address

type Address struct {
	AddressType Type
	Street      string
	House       string
	Apartment   string
}

type Type string

const (
	Fact Type = "FACT"
	Req  Type = "REG"
)
