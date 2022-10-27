package jwt

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"

	timeUtil "cxqi/common/kit/time"

	"cxqi/common/errcode"
)

const (
	// PrivatePayloadName 私有载荷名称
	PrivatePayloadName = "x_user_info"
)

// Config JWT相关配置
type Config struct {
	Issuer         string        // 签发者
	SecretKey      string        // 密钥
	ExpirationTime time.Duration // 过期时间
}

// JWT JWT结构详情
type JWT struct {
	c *Config
}

// NewJWT 新建JWT
func NewJWT(c *Config) (*JWT, error) {
	if c == nil || c.Issuer == "" || c.SecretKey == "" || c.ExpirationTime.Seconds() <= 0 {
		return nil, errors.New("jwt: illegal jwt configure")
	}

	return &JWT{c: c}, nil
}

// MustNewJWT 新建JWT
func MustNewJWT(c *Config) *JWT {
	j, err := NewJWT(c)
	if err != nil {
		panic(err)
	}

	return j
}

// CreateToken 创建JWT字符串
func (j *JWT) CreateToken(token interface{}, expirationTime ...time.Duration) (string, error) {
	et := j.c.ExpirationTime
	if len(expirationTime) > 0 && expirationTime[0].Seconds() > 0 {
		et = expirationTime[0]
	}

	payload, err := json.Marshal(token)
	if err != nil {
		return "", errors.WithMessage(err, "json marshal token err")
	}

	claims := make(jwt.MapClaims)
	// https://www.iana.org/assignments/jwt/jwt.xhtml
	// 预定义载荷
	now := timeUtil.Now()
	claims["iss"] = j.c.Issuer         // issuer，签发者
	claims["exp"] = now.Add(et).Unix() // expiration time，过期时间
	claims["iat"] = now.Unix()         // issued at，签发时间
	claims["nbf"] = now.Unix()         // not before，生效时间
	// 私有载荷
	claims[PrivatePayloadName] = base64.StdEncoding.EncodeToString(payload)

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ts, err := t.SignedString([]byte(j.c.SecretKey))
	if err != nil {
		return "", errors.WithMessage(err, "sign token err")
	}

	return ts, nil
}

// ParseToken 解析JWT字符串
func (j *JWT) ParseToken(tokenStr string, token interface{}) error {
	tokenStr = strings.Replace(tokenStr, "Bearer ", "", 1)

	t, err := jwt.Parse(tokenStr, jwtKeyFunc(j.c.SecretKey))
	if err != nil {
		if e, ok := err.(*jwt.ValidationError); ok {
			if e.Errors&jwt.ValidationErrorMalformed != 0 {
				return errcode.ErrTokenVerify
			} else if e.Errors&jwt.ValidationErrorExpired != 0 {
				return errcode.ErrTokenExpire
			} else if e.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return errcode.ErrTokenNotValidYet
			} else {
				return errcode.ErrTokenVerify
			}
		}
		return errcode.ErrTokenVerify
	}

	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok || !t.Valid {
		return errcode.ErrTokenVerify
	}

	s, ok := claims[PrivatePayloadName].(string)
	if !ok {
		return errcode.ErrTokenVerify
	}

	payload, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return errcode.ErrTokenVerify
	}

	err = json.Unmarshal(payload, token)
	if err != nil {
		return errcode.ErrTokenVerify
	}

	return nil
}

// jwtKeyFunc JWT签名密钥函数
func jwtKeyFunc(key string) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	}
}
