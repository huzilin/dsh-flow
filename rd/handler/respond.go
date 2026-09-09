// Package handler Gin 路由薄壳。本文件：protojson 出参/入参编解码 +
// 统一失败出参写出（契约真相源 docs/api/v1/api.proto）。
package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/huzilin/wbflow/rd/api"
	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// protoJSONMarshal protojson 出参编码：EmitUnpopulated 使零值字段可见
// （spec §1.1 统一响应口径——出参字段恒显式；int64 字段映射为 JSON 字符串，
// 见 api.proto 头注释取舍说明）。
var protoJSONMarshal = protojson.MarshalOptions{EmitUnpopulated: true}

// protoJSONUnmarshal protojson 入参解码：DiscardUnknown 容忍未知字段（向前兼容）。
var protoJSONUnmarshal = protojson.UnmarshalOptions{DiscardUnknown: true}

// respondProto 成功出参：protojson 序列化 + application/json。
func respondProto(c *gin.Context, status int, msg proto.Message) {
	data, err := protoJSONMarshal.Marshal(msg)
	if err != nil {
		AbortWithErr(c, errs.Trace(err))
		return
	}
	c.Data(status, "application/json; charset=utf-8", data)
}

// bindProtoBody 请求体解码到 req（proto 消息）；空请求体合法（全默认值，
// 如 advance/deprecate 不带 updated_at）。解码失败返回入参非法错误。
func bindProtoBody(c *gin.Context, req proto.Message) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return errs.Trace(err)
	}
	if len(body) == 0 {
		return nil
	}
	if err := protoJSONUnmarshal.Unmarshal(body, req); err != nil {
		return invalidArgErr("请求体非合法 JSON 契约: %v", err)
	}
	return nil
}

// AbortWithErr 错误集中映射出参：业务 handler 捕获 service 错误后统一走此写出
// （哨兵错误 → HTTP 状态码/错误码/blocked 标记，映射表见 errs.MapToHTTP）；
// 判据校验类拦截（errs.WithMissing 载体）出参附带 missing 缺项清单。
func AbortWithErr(c *gin.Context, err error) {
	mapping := errs.MapToHTTP(err)
	writeErrJSON(c, mapping.HTTPStatus, mapping.Code, err.Error(), mapping.Blocked, errs.MissingOf(err))
}

// writeErrJSON 统一失败出参写出（api.ErrorResp 契约：{ok:false, error, code, blocked, missing}）。
func writeErrJSON(c *gin.Context, status int, code errs.Code, msg string, blocked bool, missing []string) {
	if missing == nil {
		missing = make([]string, 0)
	}
	data, mErr := protoJSONMarshal.Marshal(&api.ErrorResp{
		Ok:      false,
		Error:   msg,
		Code:    string(code),
		Blocked: blocked,
		Missing: missing,
	})
	if mErr != nil {
		// 出参编码失败兜底：纯状态 500（不 panic）
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Data(status, "application/json; charset=utf-8", data)
	// 终止后续 handler 链（c.Data 不设 abort 标记，401/409 等短路出参必须显式 Abort）
	c.Abort()
}

// invalidArgErr handler 层入参非法错误（哨兵 errs.ErrInvalidArgument 为 cause）。
func invalidArgErr(format string, args ...any) error {
	return errs.Trace(fmt.Errorf("%w: %s", errs.ErrInvalidArgument, fmt.Sprintf(format, args...)))
}
