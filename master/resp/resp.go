package resp

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	OK            = NewResp(0, "成功")
	ServerError   = NewResp(10000000, "服务内部错误")
	InvalidParams = NewResp(10000001, "入参错误")
)

type resp struct {
	code int
	msg  string
}

var codes = map[int]string{}

func NewResp(code int, msg string) *resp {
	if _, ok := codes[code]; ok {
		log.Fatalf("错误码 %d 已经存在，请更换一个\n", code)
	}
	codes[code] = msg
	return &resp{code: code, msg: msg}
}

func Response(ctx *gin.Context, status int, resp *resp, data any) {
	ctx.JSON(status, gin.H{
		"code": resp.code,
		"msg":  resp.msg,
		"data": data,
	})
}

func ResponseOK(ctx *gin.Context, data any) {
	Response(ctx, http.StatusOK, OK, data)
}
