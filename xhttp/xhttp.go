package xhttp

import (
	"bytes"
	"context"
	"cxqi/common/errcode"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"

	"cxqi/common/kit/convert"

	"cxqi/common/logger/xzap"

	"cxqi/commonn/kit/validator"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

/*
   This article is a detailed standard processing of the returned structure,
   similar to the middleware library, and related functions can be added according
   to the actual situation
*/

//Response Business Generic Response Structure
type Reponse struct {
	//ink tracking id
	TraceId string `json:"trace_id" example:"a1b2c3d4e5f6g7h8" extensions:"x-order=000"`

	//status code
	Code uint32 `json:"code" example:"200" extensions:"x-order=001"`

	//information
	Msg string `json:"msg" example:"OK" extensions:"x-order=002"`

	//return data
	Data interface{} `json:"data" extensions:"x-order=003"`
}

//GetTraceId get link tracking id
func GetTraceId(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}

func WriteHeader(w http.ResponseWriter, err ...error) {
	var ee error
	if len(err) > 0 {
		ee = err[0]
	}

	e := errcode.ParseErr(ee)

	w.Header().Set(HeaderGWErrorCode, convert.ToString(e.Code()))
	w.Header().Set(HeaderGWErrorMessage, url.QueryEscape(e.Error()))
}

// Error 错误响应返回
func Error(c *gin.Context, err error) {
	ctx := c.Request.Context()
	e := errcode.ParseErr(err)
	if e == errcode.ErrUnexpected || e == errcode.ErrCustom {
		xzap.WithContext(ctx).Errorf("request handle err, code: %d, err: %+v", e.Code(), err)
	}

	WriteHeader(c.Writer, e)
	c.JSON(e.HTTPCode(), &Reponse{
		TraceId: GetTraceId(ctx),
		Code:    e.Code(),
		Msg:     e.Error(),
		Data:    nil,
	})
}

// Parse 请求体解析
func Parse(r *http.Request, v interface{}) error {
	// if err := httpx.Parse(r, v); err != nil {
	// 	xzap.WithContext(r.Context()).Errorf("request parse err, err: %s", formatStr(err.Error(), halfShowLen))
	// 	return errcode.ErrInvalidParams
	// }

	if err := validator.Verify(v); err != nil {
		return errcode.NewCustomErr(err.Error())
	}

	return nil
}

// ParseForm 请求表单解析
func ParseForm(r *http.Request, v interface{}) error {
	// if err := httpx.ParseForm(r, v); err != nil {
	// 	xzap.WithContext(r.Context()).Errorf("request parse form err, err: %s",
	// 		formatStr(err.Error(), halfShowLen))
	// 	return errcode.ErrInvalidParams
	// }

	if err := validator.Verify(v); err != nil {
		return errcode.NewCustomErr(err.Error())
	}

	return nil
}

// FromFile 请求表单文件获取
func FromFile(r *http.Request, name string, size int64) (*multipart.FileHeader, error) {
	if r.MultipartForm == nil {
		if err := r.ParseMultipartForm(size); err != nil {
			return nil, err
		}
	}

	f, fh, err := r.FormFile(name)
	if err != nil {
		if err == http.ErrMissingFile {
			return nil, errcode.ErrInvalidParams
		}
		return nil, err
	}
	f.Close()
	return fh, nil
}

// Query 返回给定请求查询参数键的字符串值
func Query(r *http.Request, key string) string {
	value, _ := GetQuery(r, key)
	return value
}

// GetQuery 返回给定请求查询参数键的字符串值并判断其是否存在
func GetQuery(r *http.Request, key string) (string, bool) {
	if values, ok := GetQueryArray(r, key); ok {
		return values[0], ok
	}
	return "", false
}

// QueryArray 返回给定请求查询参数键的字符串切片值
func QueryArray(r *http.Request, key string) []string {
	values, _ := GetQueryArray(r, key)
	return values
}

// GetQueryArray 返回给定请求查询参数键的字符串切片值并判断其是否存在
func GetQueryArray(r *http.Request, key string) ([]string, bool) {
	query := r.URL.Query()
	if values, ok := query[key]; ok && len(values) > 0 {
		return values, true
	}
	return []string{}, false
}

// GetClientIP 获取客户端的IP
func GetClientIP(r *http.Request) string {
	ip := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
	if ip != "" {
		return ip
	}

	ip = strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	if ip != "" {
		return ip
	}

	if addr := r.Header.Get("X-Appengine-Remote-Addr"); addr != "" {
		return addr
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
		return ip
	}

	return ""
}

// GetExternalIP 通过API获取服务端的外部IP
func GetExternalIP() (string, error) {
	api := "http://pv.sohu.com/cityjson?ie=utf-8"

	resp, err := http.Get(api)
	if err != nil {
		return "", errors.WithMessagef(err, "http get api = %v err", api)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.WithMessage(err, "read all response body err")
	}
	s := string(b)

	i := strings.Index(s, `"cip": "`)
	s = s[i+len(`"cip": "`):]
	i = strings.Index(s, `"`)
	s = s[:i]

	return s, nil
}

// GetInternalIP 获取服务端的内部IP
func GetInternalIP() string {
	infs, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, inf := range infs {
		if isEthDown(inf.Flags) || isLoopback(inf.Flags) {
			continue
		}

		addrs, err := inf.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
	}

	return ""
}

func isEthDown(f net.Flags) bool {
	return f&net.FlagUp != net.FlagUp
}

func isLoopback(f net.Flags) bool {
	return f&net.FlagLoopback == net.FlagLoopback
}

func formatStr(s string, halfShowLen int) string {
	if length := len(s); length > halfShowLen*2 {
		return s[:halfShowLen] + " ...... " + s[length-halfShowLen-1:]
	}

	return s
}

// CopyHttpRequest 复制请求体
func CopyHttpRequest(r *http.Request) (*http.Request, error) {
	rClone := r.Clone(context.Background())
	// 克隆请求体
	if r.Body != nil {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}

		r.Body = ioutil.NopCloser(bytes.NewReader(body))
		rClone.Body = ioutil.NopCloser(bytes.NewReader(body))
	}

	return rClone, nil
}
