package recover

import (
	"fmt"
	baseError "github.com/go-estar/base-error"
	"github.com/go-estar/config"
	"github.com/go-estar/iris/baseContext"
	"github.com/kataras/iris/v12"
	"reflect"
)

func New() iris.Handler {
	return baseContext.Handler(Recover())
}

func Recover() func(ctx *baseContext.Context) {
	return func(ctx *baseContext.Context) {
		defer func() {
			if err := recover(); err != nil {
				if ctx.IsStopped() {
					return
				}

				var e error
				switch err.(type) {
				case error:
					e = baseError.NewSystemCodeWrap(ctx.ErrorCodes["System"], err.(error), baseError.WithStack(6))
				default:
					e = baseError.NewSystemCode(ctx.ErrorCodes["System"], fmt.Sprint(err), baseError.WithStack(6))
				}

				if ctx.Env != config.Production.String() {
					console := fmt.Sprintf("Recover: %s\n", reflect.TypeOf(err))
					console += fmt.Sprintf("%s\n", ctx.HandlerName())
					console += fmt.Sprintf("%+v", e)
					ctx.Application().Logger().Error(console)
				}

				ctx.Values().Set("error", e)

				ctx.StatusCode(500)
				ctx.StopExecution()
			}
		}()

		ctx.Next()
	}

}
