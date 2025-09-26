package router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"path/filepath"
	"regexp"
	"time"
)

var tokenRegexp = regexp.MustCompile("token=[^&]+")

func logFormatter(param gin.LogFormatterParams) string {
	if (param.ClientIP == "127.0.0.1" || param.ClientIP == "::1") && param.Path == "/health" {
		return ""
	}

	var statusColor, methodColor, resetColor string
	if param.IsOutputColor() {
		statusColor = param.StatusCodeColor()
		methodColor = param.MethodColor()
		resetColor = param.ResetColor()
	}

	if param.Latency > time.Minute {
		param.Latency = param.Latency - param.Latency%time.Second
	}
	path := tokenRegexp.ReplaceAllString(param.Path, "token=[masked]")
	return fmt.Sprintf("%v |%s %3d %s| %13v | %15s |%s %-7s %s %#v\n%s",
		param.TimeStamp.Format(time.RFC3339),
		statusColor, param.StatusCode, resetColor,
		param.Latency,
		param.ClientIP,
		methodColor, param.Method, resetColor,
		path,
		param.ErrorMessage,
	)
}

func NotFound(ctx *gin.Context) {
	ctx.JSON(http.StatusNotFound, gin.H{
		"error":      fmt.Sprintf("Route %s not found", filepath.Clean(ctx.Request.URL.Path)),
		"error_code": http.StatusNotFound,
	})
}

func CreateRouter() (g *gin.Engine, exit func()) {
	g = gin.New()

	// nginx相关配置
	g.RemoteIPHeaders = []string{"X-Forwarded-For"}
	g.SetTrustedProxies(nil) // 信任所有代理
	g.ForwardedByClientIP = true

	g.Use(func(ctx *gin.Context) {
		// Map sockets "@" to 127.0.0.1, because gin-gonic can only trust IPs.
		if ctx.Request.RemoteAddr == "@" {
			ctx.Request.RemoteAddr = "127.0.0.1:65535"
		}
	})

	g.Use(gin.LoggerWithFormatter(logFormatter), gin.Recovery(), gerror.Handler(), location.Default())
	g.NoRoute(NotFound)

}
