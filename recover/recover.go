package recover

import (
	"fmt"
	"github.com/go-estar/config"
	"github.com/go-estar/iris/baseContext"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
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
					e = errors.WithStack(err.(error))
				default:
					e = errors.New(fmt.Sprint(err))
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
