package code

import (
	"net/http"
	"sync"
)

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

var codes = map[int]*Errcode{}
var codeMux = &sync.Mutex{}

func register(code int, httpStatus int, message string) {
	if code == 0 {
		panic("code must not be 0")
	}
	if _, ok := OnlyUseHTTPStatus[httpStatus]; !ok {
		panic("http status must be in OnlyUseHTTPStatus")
	}

	codeMux.Lock()
	defer codeMux.Unlock()
	errcode := &Errcode{
		code:    code,
		http:    httpStatus,
		message: message,
	}
	codes[code] = errcode
}

func ParseCoder(code int) *Errcode {
	if coder, ok := codes[code]; ok {
		return coder
	}
	return &Errcode{
		code:    1,
		http:    http.StatusInternalServerError,
		message: "unknown error",
	}
}
