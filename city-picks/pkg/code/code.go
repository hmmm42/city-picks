package code

type Errcode struct {
	code    int
	http    int
	message string
}

func (coder Errcode) Error() string {
	return coder.message
}

func (coder Errcode) Code() int {
	return coder.code
}

func (coder Errcode) String() string {
	return coder.message
}

func (coder Errcode) HTTPStatus() int {
	if coder.http == 0 {
		return 500
	}
	return coder.http
}
