package resptypes

type (
	Null struct{}
)

func (r Null) ToRespString() string {
	return "_\r\n"
}

func (r Null) toString() string {
	return ""
}
