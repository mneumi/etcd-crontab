package middleware

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/master/resp"
)

func GetIntercept() gin.HandlerFunc {
	// 如果环境变量 INTERCEPT 不为空，则开启拦截模式
	intercept := os.Getenv(common.ENV_KEY_INTERCEPT) != ""

	// 启动协程，每30秒检查一次环境变量，判断拦截模式的状态
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			intercept = os.Getenv(common.ENV_KEY_INTERCEPT) != ""
		}
	}()

	return func(ctx *gin.Context) {
		if intercept {
			resp.Response(ctx, http.StatusInternalServerError, resp.InterceptMode, "拦截模式开启")
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

func GetCORS() gin.HandlerFunc {
	return cors.Default()
}
