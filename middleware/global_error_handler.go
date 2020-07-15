package middleware

import (
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/m2c/kiplestar/commons"
	cerror "github.com/m2c/kiplestar/commons/error"
	slog "github.com/m2c/kiplestar/commons/log"
	"runtime"
)

/**
* @Description: handler error infor
* @Author: seven
* @Date: 2019/10/23
 */
func GlobalRecover(ctx iris.Context) {
	defer func() {
		if err := recover(); err != nil {
			if ctx.IsStopped() {
				return
			}
			switch err.(type) {
			case *cerror.CommonsError:
				{
					commonsError := err.(*cerror.CommonsError)
					code := commonsError.StatusCode
					stack := commonsError.CallStack()
					slog.Error(stack)
					msg := commons.BuildFailedWithMsg(code, commonsError.Message)
					ctx.JSON(msg)
				}
			default:
				{
					var stacktrace string
					for i := 1; ; i++ {
						_, f, l, got := runtime.Caller(i)
						if !got {
							break
						}
						stacktrace += fmt.Sprintf("%s:%d\n", f, l)
					}
					slog.Error(stacktrace)
					ctx.JSON(commons.BuildFailed(commons.UnKnowError))
				}
			}
			ctx.StopExecution()
		}
	}()
	ctx.Next()
}
