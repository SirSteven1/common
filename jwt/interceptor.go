package jwt

import (
	"context"

	"cxqi/common/kit/convert"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// TokenKey 用户令牌key
	TokenKey = contextKey("X-Token")

	// TokenTypeAccess 令牌类型：访问
	TokenTypeAccess = "access"
	// TokenTypeRefresh 令牌类型：刷新
	TokenTypeRefresh = "refresh"

	// LoginTypeWallet 登录类型：钱包地址
	LoginTypeWallet = "wallet"
	// LoginTypeEmail 登录类型：邮箱
	LoginTypeEmail = "email"

	tokenTypeKey = "X-Token-Type"
	randomIdKey  = "X-Random-Id"
	loginTypeKey = "X-Login-Type"
	userIdKey    = "X-User-Id"
	roleIdKey    = "X-Role-Id"
)

// contextKey 上下文key类型
type contextKey string

// String 实现序列化字符串方法
func (c contextKey) String() string {
	return "jwt context key: " + string(c)
}

// Token 默认令牌结构详情
type Token struct {
	TokenType string  `json:"token_type"` // 令牌类型，固定为access
	RandomId  string  `json:"random_id"`  // 随机id
	LoginType string  `json:"login_type"` // 登录类型
	UserId    int64   `json:"user_id"`    // 用户id
	RoleIds   []int64 `json:"role_ids"`   // 角色id列表
}

// Visit 访问并操作默认令牌数据
func (t *Token) Visit(fn func(key, val string) bool) {
	fn(tokenTypeKey, t.TokenType)
	fn(randomIdKey, t.RandomId)
	fn(loginTypeKey, t.LoginType)
	fn(userIdKey, convert.ToString(t.UserId))
	for _, roleId := range t.RoleIds {
		fn(roleIdKey, convert.ToString(roleId))
	}
}

// WithToken 将默认令牌数据关联到context中
func WithToken(ctx context.Context, token *Token) context.Context {
	return context.WithValue(ctx, TokenKey, token)
}

// FromContext 从context获取默认令牌数据
func FromContext(ctx context.Context) (*Token, bool) {
	if token, ok := ctx.Value(TokenKey).(*Token); ok {
		return token, true
	}

	return nil, false
}

// FromMD 从grpc metadata获取默认令牌数据
func FromMD(md metadata.MD) (*Token, bool) {
	if md == nil {
		return nil, false
	}

	var t Token

	if tts := md.Get(tokenTypeKey); len(tts) > 0 {
		t.TokenType = tts[0]
	}
	if rais := md.Get(randomIdKey); len(rais) > 0 {
		t.RandomId = rais[0]
	}
	if lts := md.Get(loginTypeKey); len(lts) > 0 {
		t.LoginType = lts[0]
	}
	if uis := md.Get(userIdKey); len(uis) > 0 {
		t.UserId = convert.ToInt64(uis[0])
	}
	if ris := md.Get(roleIdKey); len(ris) > 0 {
		roleIds := make([]int64, 0, len(ris))
		for _, ri := range ris {
			roleIds = append(roleIds, convert.ToInt64(ri))
		}
		t.RoleIds = roleIds
	}

	return &t, true
}

// TokenInterceptor 默认令牌服务端一元拦截器
func TokenInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(wrapServerContext(ctx), req)
}

// TokenStreamInterceptor 默认令牌服务端流拦截器
func TokenStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	wss := newWrappedServerStream(ss)
	wss.WrappedContext = wrapServerContext(wss.WrappedContext)

	return handler(srv, wss)
}

// TokenClientInterceptor 默认令牌客户端一元拦截器
func TokenClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return invoker(wrapClientContext(ctx), method, req, reply, cc, opts...)
}

// TokenStreamClientInterceptor 默认令牌客户端流拦截器
func TokenStreamClientInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return streamer(wrapClientContext(ctx), desc, cc, method, opts...)
}

// wrapServerContext 包装服务端上下文
func wrapServerContext(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	token, ok := FromMD(md)
	if !ok {
		return ctx
	}

	return context.WithValue(ctx, TokenKey, token)
}

// wrapClientContext 包装客户端上下文
func wrapClientContext(ctx context.Context) context.Context {
	if token, ok := ctx.Value(TokenKey).(*Token); ok {
		var pairs []string
		token.Visit(func(key, val string) bool {
			pairs = append(pairs, key, val)
			return true
		})

		ctx = metadata.AppendToOutgoingContext(ctx, pairs...)
	}

	return ctx
}

// wrappedServerStream 包装后的服务端流对象
type wrappedServerStream struct {
	grpc.ServerStream
	WrappedContext context.Context
}

// newWrappedServerStream 新建包装后的服务端流对象
func newWrappedServerStream(ss grpc.ServerStream) *wrappedServerStream {
	if existing, ok := ss.(*wrappedServerStream); ok {
		return existing
	}
	return &wrappedServerStream{ServerStream: ss, WrappedContext: ss.Context()}
}

// Context 返回包装后的服务端流对象的上下文信息
func (w *wrappedServerStream) Context() context.Context {
	return w.WrappedContext
}
