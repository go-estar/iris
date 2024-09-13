package anyError

import (
	"fmt"
	baseError "github.com/go-estar/base-error"
	"github.com/go-estar/iris/baseContext"
	"github.com/kataras/iris/v12"
	"net/http"
)

func New() iris.Handler {
	return baseContext.Handler(AnyError)
}

func AnyError(ctx *baseContext.Context) {
	text := http.StatusText(ctx.GetStatusCode())
	var err = ctx.GetErr()
	if err == nil {
		err = baseError.New(fmt.Sprintf("%d %s", ctx.GetStatusCode(), text))
	}
	if ctx.GetContentTypeRequested() != "" {
		ctx.Error(err)
	} else {
		ctx.SetErr(err)
		ctx.WriteString(text)
	}
	ctx.Logger.Log(ctx)
}
