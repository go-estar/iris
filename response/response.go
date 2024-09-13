package response

import (
	"github.com/go-estar/iris/baseContext"
	"github.com/go-estar/types/fieldUtil"
)

func New() baseContext.Response {
	return &Response{}
}

type Response struct {
	Code    any         `json:"code"`
	Message string      `json:"message"`
	System  bool        `json:"system,omitempty"`
	Chain   []string    `json:"chain,omitempty"`
	Rid     interface{} `json:"rid,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func (r *Response) Success() baseContext.Response {
	r.Code = "00"
	return r
}

func (r *Response) SetCode(code string) baseContext.Response {
	r.Code = code
	return r
}

func (r *Response) SetMessage(message string) baseContext.Response {
	r.Message = message
	return r
}

func (r *Response) SetData(data interface{}) baseContext.Response {
	if !fieldUtil.IsNil(data) {
		r.Data = data
	}
	return r
}

func (r *Response) SetSystem() baseContext.Response {
	r.System = true
	return r
}

func (r *Response) SetChain(chain ...string) baseContext.Response {
	r.Chain = append(r.Chain, chain...)
	return r
}

func (r *Response) SetRid(rid string) baseContext.Response {
	r.Rid = rid
	return r
}

func (r *Response) ContentType() string {
	return "json"
}

func (r *Response) Content() interface{} {
	return r
}
