package jwt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestFromMD(t *testing.T) {
	token := &Token{
		TokenType: "access",
		RandomId:  "abcdefgh",
		LoginType: LoginTypeWallet,
		UserId:    1000,
		RoleIds:   []int64{4000, 5000, 6000},
	}

	var pairs []string
	token.Visit(func(key, val string) bool {
		pairs = append(pairs, key, val)
		return true
	})

	md := metadata.Pairs(pairs...)
	t.Log(md)

	parseToken, ok := FromMD(md)
	assert.True(t, ok)
	if assert.Equal(t, token, parseToken) {
		t.Logf("%+v", parseToken)
	}
}
