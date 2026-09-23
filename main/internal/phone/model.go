package phone

type Phone struct {
	Type   Type
	Number string
}

type Type string

const (
	Home   Type = "HOME"
	Work   Type = "WORK"
	Mobile Type = "MOBILE"
)
