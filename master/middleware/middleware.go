package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/mneumi/etcd-crontab/master/resp"
)

func GetIntercept(intercept *bool) gin.HandlerFunc {

	return func(ctx *gin.Context) {
		method := ctx.Request.Method
		if *intercept {
			if method == "POST" || method == "PUT" || method == "DELETE" {
				resp.ResponseOK(ctx, "拦截模式开启")
				ctx.Abort()
				return
			}
		}
		ctx.Next()
	}
}

func GetCORS() gin.HandlerFunc {
	return cors.Default()
}
