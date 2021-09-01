package middleware

import (
	"context"
	"github.com/kataras/iris/v12"
	"github.com/m2c/kiplestar/commons"
	slog "github.com/m2c/kiplestar/commons/log"
	uuid "github.com/satori/go.uuid"
)

func TraceLogger(ctx iris.Context) {
	requestID := ctx.Request().Header.Get(commons.X_REQUEST_ID)
	if len(requestID) == 0 {
		requestID = uuid.NewV4().String()
	}
	traceContext := context.WithValue(ctx.Request().Context(), commons.X_REQUEST_ID, requestID)
	newRequest := ctx.Request().WithContext(traceContext)
	ctx.ResetRequest(newRequest)
	path := ctx.Request().URL.Path
	method := ctx.Request().Method
	ip := ctx.Request().RemoteAddr
	slog.InfofStdCtx(traceContext, "path:%s method:%s ip:%s start \n", requestID, path, method, ip)
	ctx.Next()
	slog.InfofStdCtx(traceContext, "done \n")
}
