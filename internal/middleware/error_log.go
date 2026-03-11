package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/services"
)

type bodyCaptureWriter struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	// 限制缓存大小，避免超大响应占内存
	const max = 8 * 1024
	remain := max - w.buf.Len()
	if remain > 0 {
		if len(b) <= remain {
			_, _ = w.buf.Write(b)
		} else {
			_, _ = w.buf.Write(b[:remain])
		}
	}
	return w.ResponseWriter.Write(b)
}

func extractErrorMessage(body []byte) string {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return ""
	}
	// 仅尝试解析 {"error":"..."} 结构
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return ""
	}
	if v, ok := obj["error"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ErrorLog 捕获 panic 与 HTTP 错误响应，落库记录报错信息/位置/时间
func ErrorLog(errSvc *services.ErrorLogService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// panic 兜底
		defer func() {
			if r := recover(); r != nil {
				msg := "panic"
				switch v := r.(type) {
				case string:
					msg = v
				case error:
					msg = v.Error()
				}
				uid := GetUserID(c)
				var userID *uint
				if uid != 0 {
					userID = &uid
				}
				_ = errSvc.Create(services.CreateErrorLogRequest{
					UserID:   userID,
					Level:    "panic",
					Message:  msg,
					Location: c.Request.Method + " " + c.FullPath(),
					Method:   c.Request.Method,
					Path:     c.Request.URL.Path,
					Status:   http.StatusInternalServerError,
					Stack:    string(debug.Stack()),
				})
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			}
		}()

		// 捕获响应 body（用于抽取 error 字段）
		bw := &bodyCaptureWriter{ResponseWriter: c.Writer}
		c.Writer = bw

		c.Next()

		status := c.Writer.Status()
		if status < 400 {
			return
		}

		// 仅记录带 error 字段或 5xx
		body := bw.buf.Bytes()
		errMsg := extractErrorMessage(body)
		if errMsg == "" && status < 500 {
			return
		}
		if errMsg == "" {
			errMsg = http.StatusText(status)
		}

		uid := GetUserID(c)
		var userID *uint
		if uid != 0 {
			userID = &uid
		}
		loc := c.Request.Method + " " + c.Request.URL.Path
		if p := c.FullPath(); strings.TrimSpace(p) != "" {
			loc = c.Request.Method + " " + p
		}
		_ = errSvc.Create(services.CreateErrorLogRequest{
			UserID:   userID,
			Level:    "error",
			Message:  errMsg,
			Location: loc,
			Method:   c.Request.Method,
			Path:     c.Request.URL.Path,
			Status:   status,
			Stack:    "",
		})
	}
}

