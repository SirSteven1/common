package errcode

import (
	"errors"
	"net/http"

	"google.golang.org/grpc/status"
)

const (

	//CodeOK Request successful business status code
	CodeOK = http.StatusOK

	//MsgOK request success message
	MsgOK = "Successful"

	//CodeCustom custom error business status code
	CodeCustom = 7000

	//MsgCustom custom error message
	MsgCustom = "Custom error"
)

//Err business error structure
type Err struct {
	code     uint32
	httpCode int
	msg      string
}

//Code business status code
func (e *Err) Code() uint32 {
	return e.code
}

//HTTPCode HTTP status code
func (e *Err) HTTPCode() int {
	return e.httpCode
}

//Error message
func (e *Err) Error() string {
	return e.msg
}

//business error
var (
	NoErr = NewErr(CodeOK, MsgOK)

	ErrCustom     = NewErr(CodeCustom, MsgCustom)
	ErrUnexpected = NewErr(7777, "Network error, please try again later", http.StatusInternalServerError)

	ErrTokenNotValidYet    = NewErr(9999, "Token illegal", http.StatusUnauthorized)
	ErrInvalidUrl          = NewErr(10000, "URL is illegal")
	ErrInvalidHeader       = NewErr(10001, "Invalid request header")
	ErrInvalidParams       = NewErr(10002, "Parameter is illegal")
	ErrTokenVerify         = NewErr(10003, "Token check error", http.StatusUnauthorized)
	ErrTokenExpire         = NewErr(10004, "Expired token", http.StatusUnauthorized)
	ErrUserLogin           = NewErr(10005, "User log out", http.StatusUnauthorized)
	ErrUserPrivilegeChange = NewErr(10006, "Permission changed", http.StatusUnauthorized)
	ErrLockNotAcquire      = NewErr(10007, "Lock not released")
	ErrLockAcquire         = NewErr(10008, "Lock acquisition error")
	ErrLockNotRelease      = NewErr(10009, "Lock is not released")
	ErrLockRelease         = NewErr(10010, "Lock released err")
	ErrTWitterAddress      = NewErr(10011, "Twitter address illeage")
	ErrDiscordAddress      = NewErr(10012, "Discord address illeage")
	ErrAddress             = NewErr(10013, "Address illeage")
)

var codeToErr = map[uint32]*Err{
	200: NoErr,

	7000: ErrCustom,
	7777: ErrUnexpected,

	9999:  ErrTokenNotValidYet,
	10000: ErrInvalidUrl,
	10001: ErrInvalidHeader,
	10002: ErrInvalidParams,
	10003: ErrTokenVerify,
	10004: ErrTokenExpire,
	10005: ErrUserLogin,
	10006: ErrUserPrivilegeChange,
	10007: ErrLockNotAcquire,
	10008: ErrLockAcquire,
	10009: ErrLockNotRelease,
	10010: ErrLockRelease,
	10011: ErrTWitterAddress,
	10012: ErrDiscordAddress,
	10013: ErrAddress,
}

//NewErr creates a new business error
func NewErr(code uint32, msg string, httpCode ...int) *Err {
	hc := http.StatusOK
	if len(httpCode) != 0 {
		hc = httpCode[0]
	}
	return &Err{code: code, httpCode: hc, msg: msg}
}

func GetCodeToErr() map[uint32]*Err {
	return codeToErr
}

func SetCodeToErr(code uint32, err *Err) error {
	if _, ok := codeToErr[code]; ok {
		return errors.New("has exist")
	}

	codeToErr[code] = err
	return nil
}

// NewCustomErr creates a new custom error
func NewCustomErr(msg string, httpCode ...int) *Err {
	return NewErr(CodeCustom, msg, httpCode...)
}

// IsErr judges whether it is a business error
func IsErr(err error) bool {
	if err == nil {
		return true
	}

	_, ok := err.(*Err)
	return ok
}

// ParseErr parsing business errors
func ParseErr(err error) *Err {
	if err == nil {
		return NoErr
	}

	if e, ok := err.(*Err); ok {
		return e
	}

	s, _ := status.FromError(err)
	c := uint32(s.Code())
	if c == CodeCustom {
		return NewCustomErr(s.Message())
	}

	return ParseCode(c)
}

// ParseCode parses the business error corresponding to the business status code
func ParseCode(code uint32) *Err {
	if e, ok := codeToErr[code]; ok {
		return e
	}

	return ErrUnexpected
}
