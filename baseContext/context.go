package baseContext

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	baseError "github.com/go-estar/base-error"
	"github.com/go-estar/config"
	localTime "github.com/go-estar/local-time"
	"github.com/go-estar/logger"
	"github.com/go-estar/types/fieldUtil"
	"github.com/go-estar/types/jsonUtil"
	"github.com/go-estar/validate"
	"github.com/go-playground/validator/v10"
	"github.com/iris-contrib/schema"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/sessions"
	"reflect"
	"strings"
)

var (
	ErrorSystem     = "100"
	ErrorReadParams = "101"
	ErrorValidation = "102"
)

type Logger interface {
	GetLogger() logger.Logger
	Handler() iris.Handler
	Log(*Context)
}

type Response interface {
	Success() Response
	SetCode(code string) Response
	SetMessage(message string) Response
	SetData(data interface{}) Response
	SetSystem() Response
	SetChain(chain ...string) Response
	SetRid(rid string) Response
	ContentType() string
	Content() interface{}
}

type Option func(*Context)

func New(env string, logger Logger, opts ...Option) {
	baseContext = &Context{
		Env:    env,
		Logger: logger,
	}
	for _, apply := range opts {
		apply(baseContext)
	}
	if baseContext.ErrorCodes == nil {
		baseContext.ErrorCodes = make(map[string]string)
	}
	if baseContext.ErrorCodes["System"] == "" {
		baseContext.ErrorCodes["System"] = ErrorSystem
	}
	if baseContext.ErrorCodes["ReadParams"] == "" {
		baseContext.ErrorCodes["ReadParams"] = ErrorReadParams
	}
	if baseContext.ErrorCodes["Validation"] == "" {
		baseContext.ErrorCodes["Validation"] = ErrorValidation
	}
}

func WithApplicationName(val string) Option {
	return func(opts *Context) {
		opts.ApplicationName = val
	}
}

func WithResponse(val NewResponse) Option {
	return func(opts *Context) {
		opts.Response = val
	}
}

func WithViewError(val string) Option {
	return func(opts *Context) {
		opts.ViewError = val
	}
}

func WithSystemErrorCode(val string) Option {
	return func(opts *Context) {
		if opts.ErrorCodes == nil {
			opts.ErrorCodes = make(map[string]string)
		}
		opts.ErrorCodes["System"] = val
	}
}

func WithReadParamsErrorCode(val string) Option {
	return func(opts *Context) {
		if opts.ErrorCodes == nil {
			opts.ErrorCodes = make(map[string]string)
		}
		opts.ErrorCodes["ReadParams"] = val
	}
}

func WithValidationErrorCode(val string) Option {
	return func(opts *Context) {
		if opts.ErrorCodes == nil {
			opts.ErrorCodes = make(map[string]string)
		}
		opts.ErrorCodes["Validation"] = val
	}
}

type NewResponse func() Response

type Context struct {
	iris.Context
	Env             string
	ApplicationName string
	Logger          Logger
	Response        NewResponse
	ViewError       string
	ErrorCodes      map[string]string
}

const irisSessionContextKey = "iris.session"

func (ctx *Context) GetSession() *sessions.Session {
	if v := ctx.Values().Get(irisSessionContextKey); v != nil {
		if sess, ok := v.(*sessions.Session); ok {
			return sess
		}
	}
	return nil
}

func UnmarshalJSON(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func (ctx *Context) ReadJSONUseNumber(p interface{}) error {
	if err := ctx.UnmarshalBody(p, iris.UnmarshalerFunc(jsonUtil.UnmarshalUseNumber)); err != nil {
		return err
	}
	return nil
}

func (ctx *Context) JSONReqBody(p interface{}) error {
	if err := ctx.UnmarshalBody(p, iris.UnmarshalerFunc(UnmarshalJSON)); err != nil {
		return err
	}
	if reflect.TypeOf(p).Kind() == reflect.Struct || (reflect.TypeOf(p).Kind() == reflect.Ptr && reflect.TypeOf(p).Elem().Kind() == reflect.Struct) {
		if err := validate.Validate.Struct(p); err != nil {
			return baseError.NewCodeWrap(ctx.ErrorCodes["Validation"], err)
		}
	}
	return nil
}

func (ctx *Context) JSONReqForm(p interface{}) error {
	if err := ctx.ReadForm(p); err != nil {
		return err
	}
	if reflect.TypeOf(p).Kind() == reflect.Struct || (reflect.TypeOf(p).Kind() == reflect.Ptr && reflect.TypeOf(p).Elem().Kind() == reflect.Struct) {
		if err := validate.Validate.Struct(p); err != nil {
			return baseError.NewCodeWrap(ctx.ErrorCodes["Validation"], err)
		}
	}
	return nil
}

func (ctx *Context) GetResponse() Response {
	if v := ctx.Values().Get("response"); v != nil {
		response, ok := v.(NewResponse)
		if ok {
			return response()
		}
	}
	return ctx.Response()
}

func (ctx *Context) NewResponse(code string, message string, data ...interface{}) Response {
	resp := ctx.GetResponse().SetCode(code).SetMessage(message)
	if len(data) > 0 && !fieldUtil.IsNil(data[0]) {
		resp.SetData(data[0])
	}
	return resp
}

func (ctx *Context) NewSuccess(message string, data ...interface{}) Response {
	resp := ctx.GetResponse().Success().SetMessage(message)
	if len(data) > 0 && !fieldUtil.IsNil(data[0]) {
		resp.SetData(data[0])
	}
	return resp
}

func (ctx *Context) Success(data interface{}) {
	if !ctx.IsStopped() {
		ctx.StopExecution()
	}
	var resp Response
	switch v := data.(type) {
	case Response:
		resp = v
	}
	if resp == nil {
		resp = ctx.NewSuccess("", data)
	}

	if requestId := ctx.Values().GetString("requestId"); requestId != "" {
		resp.SetRid(requestId)
	}
	if resp.ContentType() == "text" {
		ctx.Text(resp.Content().(string))
	} else if resp.ContentType() == "binary" {
		ctx.Binary(resp.Content().([]byte))
	} else {
		ctx.JSON(resp.Content())
	}
}

func (ctx *Context) Error(err error, data ...interface{}) {
	if !ctx.IsStopped() {
		ctx.StopExecution()
	}
	e := ctx.BaseError(err)
	message := e.Msg
	if e.System && ctx.Env == config.Production.String() {
		message = "system error"
	}
	resp := ctx.NewResponse(e.Code, message, data...)
	if requestId := ctx.Values().GetString("requestId"); requestId != "" {
		resp.SetRid(requestId)
	}
	if e.System {
		resp.SetSystem()
	}
	if len(e.Chain) > 0 {
		resp.SetChain(e.Chain...)
	}
	if resp.ContentType() == "text" {
		ctx.Text(resp.Content().(string))
	} else if resp.ContentType() == "binary" {
		ctx.Binary(resp.Content().([]byte))
	} else {
		ctx.JSON(resp.Content())
	}
}

func (ctx *Context) ErrorView(err error, data ...interface{}) {
	if !ctx.IsStopped() {
		ctx.StopExecution()
	}
	e := ctx.BaseError(err)
	message := e.Msg
	if e.System && ctx.Env == config.Production.String() {
		message = "system error"
	}
	if requestId := ctx.Values().GetString("requestId"); requestId != "" {
		message += " rid:" + requestId
	}
	ctx.ViewData("Message", fmt.Sprintf("[%s]%s", e.Code, message))
	ctx.View(ctx.ViewError)
}

func (ctx *Context) BaseError(err error) (e *baseError.Error) {
	if ctx.GetErr() == nil {
		ctx.SetErr(err)
	}
	var errorType = reflect.TypeOf(err).String()
	//baseError
	if errorType == "*baseError.Error" {
		e = err.(*baseError.Error)
		if e.Code == "" {
			e.SetCode(ctx.ErrorCodes["System"])
		}
		if e.System || e.Stack() != nil {
			if ctx.Env != config.Production.String() {
				console := fmt.Sprintf("Error: %s\n", errorType)
				console += fmt.Sprintf("%+v", err)
				ctx.Application().Logger().Error(console)
			}
		}
		return e
	}

	//参数校验失败
	if errorType == "*json.SyntaxError" || errorType == "validator.ValidationErrors" || errorType == "schema.MultiError" {
		if errorType == "*json.SyntaxError" {
			e = baseError.NewCodeWrap(ctx.ErrorCodes["ReadParams"], err)
		} else if errorType == "validator.ValidationErrors" {
			emap := err.(validator.ValidationErrors).Translate(validate.Validate.Trans)
			e = baseError.NewCodeWrap(ctx.ErrorCodes["Validation"], errors.New(fmt.Sprint(emap)))
		} else {
			e = baseError.NewCodeWrap(ctx.ErrorCodes["ReadParams"], err)
		}
		return e
	}

	//其他错误
	if ctx.Env != config.Production.String() {
		console := fmt.Sprintf("Error: %s\n", errorType)
		console += fmt.Sprintf("%+v", err)
		ctx.Application().Logger().Error(console)
	}
	return baseError.NewSystemCodeWrap(ctx.ErrorCodes["System"], err)
}

func (ctx *Context) GetIP() string {
	ip := ctx.RemoteAddr()
	if ctx.GetHeader("X-REAL-IP") != "" {
		ip = ctx.GetHeader("X-REAL-IP")
	}
	return ip
}

func (ctx *Context) GetRequestURI() string {
	scheme := ctx.Request().URL.Scheme
	if scheme == "" {
		if ctx.Request().TLS != nil {
			scheme = "https:"
		} else {
			scheme = "http:"
		}
	}
	return scheme + "//" + ctx.Host() + ctx.Request().RequestURI
}

func (ctx *Context) GetQuery(name string) string {
	var m = make(map[string]string)
	arr := strings.Split(ctx.Request().URL.RawQuery, "&")
	for _, temp := range arr {
		index := strings.Index(temp, "=")
		if index == -1 {
			continue
		}
		key := temp[0:index]
		value := temp[index+1:]
		m[key] = value
	}
	return m[name]
}

func (ctx *Context) LogField(key string, value interface{}) *logger.Field {
	return logger.NewField(key, value)
}

func (ctx *Context) AddLogField(key string, value interface{}) {
	logFields := ctx.GetLogFields()
	logFields = append(logFields, logger.NewField(key, value))
	ctx.Values().Set("logFields", logFields)
}

func (ctx *Context) AddLogFields(fields ...*logger.Field) {
	logFields := ctx.GetLogFields()
	logFields = append(logFields, fields...)
	ctx.Values().Set("logFields", logFields)
}

func (ctx *Context) GetLogFields() []*logger.Field {
	ctxLog := ctx.Values().Get("logFields")
	if ctxLog != nil {
		logFields, ok := ctxLog.([]*logger.Field)
		if ok {
			return logFields
		}
	}
	var logFields = make([]*logger.Field, 0)
	ctx.Values().Set("logFields", logFields)
	return logFields
}

func (ctx *Context) AddLogSessionKeys(keys ...string) {
	ctx.Values().Set("logSessionKeys", keys)
}

func (ctx *Context) GetLogSessionKeys() []string {
	keys := ctx.Values().Get("logSessionKeys")
	if keys != nil {
		v, ok := keys.([]string)
		if ok {
			return v
		}
	}
	return nil
}

func (ctx *Context) AddLogContextKeys(keys ...string) {
	ctx.Values().Set("logContextKeys", keys)
}

func (ctx *Context) GetLogContextKeys() []string {
	keys := ctx.Values().Get("logContextKeys")
	if keys != nil {
		v, ok := keys.([]string)
		if ok {
			return v
		}
	}
	return nil
}

func (ctx *Context) RequestCtx() context.Context {
	return ctx.Request().Context()
}

func (ctx *Context) TraceCtx() context.Context {
	if v := ctx.Values().Get("traceCtx"); v != nil {
		traceCtx, ok := v.(context.Context)
		if ok {
			return traceCtx
		}
	}
	return context.Background()
}

func (ctx *Context) SetTraceCtx(traceCtx context.Context) {
	ctx.Values().Set("traceCtx", traceCtx)
}

func (ctx *Context) SetResponse(response NewResponse) {
	ctx.Values().Set("response", response)
}

var formDecoder *schema.Decoder

func (ctx *Context) ReadForm(p interface{}) error {
	values := ctx.FormValues()
	if len(values) == 0 {
		return nil
	}

	if reflect.TypeOf(p).Kind() == reflect.Map || (reflect.TypeOf(p).Kind() == reflect.Ptr && reflect.TypeOf(p).Elem().Kind() == reflect.Map) {
		pV := reflect.ValueOf(p)
		if pV.Kind() == reflect.Ptr {
			pV = pV.Elem()
		}
		if pV.IsZero() {
			pV.Set(reflect.MakeMap(pV.Type()))
		}
		for k, v := range values {
			if len(v) == 1 {
				pV.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v[0]))
			} else {
				pV.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
			}
		}
		return nil
	}

	if formDecoder == nil {
		formDecoder = schema.NewDecoder()
		formDecoder.IgnoreUnknownKeys(true)
		formDecoder.RegisterConverter(localTime.Time{}, func(value string) reflect.Value {
			now, err := localTime.ParseLocal(value)
			if err != nil {
				return reflect.Value{}
			}
			return reflect.ValueOf(now)
		})
	}

	return formDecoder.Decode(p, values)
}
