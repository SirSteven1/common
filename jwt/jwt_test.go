package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWT_CreateToken(t *testing.T) {
	c := &Config{Issuer: "gate-micro", SecretKey: "", ExpirationTime: 72 * time.Hour}
	_, err := NewJWT(c)
	assert.EqualError(t, err, "jwt: illegal jwt configure")

	c = &Config{Issuer: "gate-micro", SecretKey: "ABCDEFGH", ExpirationTime: -72 * time.Hour}
	_, err = NewJWT(c)
	assert.EqualError(t, err, "jwt: illegal jwt configure")

	c = &Config{Issuer: "gate-micro", SecretKey: "ABCDEFGH", ExpirationTime: 72 * time.Hour}
	j, err := NewJWT(c)
	require.NoError(t, err)

	type testToken struct {
		UserId int64 `json:"user_id"`
		RoleId int64 `json:"role_id"`
	}
	token := &testToken{UserId: 10000000, RoleId: 1}
	tokenStr, err := j.CreateToken(token)
	if assert.NoError(t, err) {
		t.Log(tokenStr)
	}

	parseToken := &testToken{}
	err = j.ParseToken(tokenStr, parseToken)
	assert.NoError(t, err)
	if assert.Equal(t, token, parseToken) {
		t.Logf("%+v", parseToken)
	}
}

func TestJWT_ParseToken(t *testing.T) {
	c := &Config{Issuer: "gate-micro", SecretKey: "", ExpirationTime: 72 * time.Hour}
	_, err := NewJWT(c)
	assert.EqualError(t, err, "jwt: illegal jwt configure")

	c = &Config{Issuer: "gate-micro", SecretKey: "ABCDEFGH", ExpirationTime: -72 * time.Hour}
	_, err = NewJWT(c)
	assert.EqualError(t, err, "jwt: illegal jwt configure")

	c = &Config{Issuer: "gate-micro", SecretKey: "ABCDEFGH", ExpirationTime: 72 * time.Hour}
	j, err := NewJWT(c)
	require.NoError(t, err)

	type testToken struct {
		UserId int64 `json:"user_id"`
		RoleId int64 `json:"role_id"`
	}
	token := &testToken{UserId: 10000000, RoleId: 1}
	tokenStr, err := j.CreateToken(token)
	if assert.NoError(t, err) {
		t.Log(tokenStr)
	}

	parseToken := &testToken{}
	err = j.ParseToken(tokenStr, parseToken)
	assert.NoError(t, err)
	if assert.Equal(t, token, parseToken) {
		t.Logf("%+v", parseToken)
	}
}
