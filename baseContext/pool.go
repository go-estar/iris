package baseContext

import (
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"reflect"
	"sync"
)

var baseContext *Context

var contextPool = sync.Pool{New: func() interface{} {
	return &Context{
		Env:             baseContext.Env,
		ApplicationName: baseContext.ApplicationName,
		Logger:          baseContext.Logger,
		Response:        baseContext.Response,
		ViewError:       baseContext.ViewError,
		ErrorCodes:      baseContext.ErrorCodes,
	}
}}

func acquire(original iris.Context) *Context {
	ctx := contextPool.Get().(*Context)
	ctx.Context = original
	return ctx
}

func release(ctx *Context) {
	contextPool.Put(ctx)
}

func Handler(h func(*Context)) iris.Handler {
	return func(original iris.Context) {
		ctx := acquire(original)
		h(ctx)
		release(ctx)
	}
}

func TypeHandler[T any](h func(*Context) (*T, error)) iris.Handler {
	return func(original iris.Context) {
		ctx := acquire(original)
		data, err := h(ctx)
		if err != nil {
			ctx.Error(err, data)
		} else {
			ctx.Success(data)
		}
		release(ctx)
	}
}

func AnyHandler(h func(*Context) (interface{}, error)) iris.Handler {
	return func(original iris.Context) {
		ctx := acquire(original)
		data, err := h(ctx)
		if err != nil {
			ctx.Error(err, data)
		} else {
			ctx.Success(data)
		}
		release(ctx)
	}
}

func call(ctx *Context, controller interface{}, methodName string) (data interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.New(fmt.Sprint(e))
		}
	}()

	controllerV := reflect.ValueOf(controller)
	method := controllerV.MethodByName(methodName)
	result := method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
	})

	if len(result) != 2 {
		err = errors.New(reflect.TypeOf(controller).String() + " " + methodName + " 返回值错误")
		return
	}

	if result[1].Interface() == nil {
		return result[0].Interface(), nil
	} else {
		err, ok := result[1].Interface().(error)
		if !ok {
			return nil, errors.New(reflect.TypeOf(controller).String() + " " + methodName + " 异常")
		}
		return result[0].Interface(), err
	}
}

func RefAnyHandler(controller interface{}, method string) iris.Handler {
	if reflect.ValueOf(controller).Kind() != reflect.Ptr {
		panic("controller must be ptr")
	}
	return func(original iris.Context) {
		ctx := acquire(original)
		data, err := call(ctx, controller, method)
		if err != nil {
			ctx.Error(err, data)
		} else {
			ctx.Success(data)
		}
		release(ctx)
	}
}
