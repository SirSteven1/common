package xkv

const (
	// CacheLaunchpadCorporationInfo 公司信息缓存
	// Cache:ServiceName:KeyPre 缓存key定义规范
	CacheLaunchpadCorporationInfo = "cache:launchpad:corporation"

	// CacheAdminMemberTokenPrefix 成员令牌数据缓存key前缀
	CacheAdminMemberTokenPrefix = "cache:admin:member_token:"

	// Lock:ServiceName:KeyPre 分布式锁key定义规范

	// LimitNotifyEmailSubscribePrefix 订阅邮件key前缀
	// Limit:ServiceName:KeyPre 限流器key定义规范
	LimitNotifyEmailSubscribePrefix = "limit:notify:email:"
)
