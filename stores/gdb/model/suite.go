package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	timeUtil "cxqi/common/kit/time"
)

// BaseSuite 基础测试套
type BaseSuite struct {
	suite.Suite
	SqlDB   *sql.DB
	SqlMock sqlmock.Sqlmock
	GormDB  *gorm.DB
}

// SetupSuite 初始化测试套
func (s *BaseSuite) SetupSuite() {
	sqlDB, sqlMock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}

	mc := mysql.Config{
		Conn:                      sqlDB,
		DefaultStringSize:         255,  // string类型字段默认长度
		DisableDatetimePrecision:  true, // 禁用datetime精度
		DontSupportRenameIndex:    true, // 禁用重命名索引
		DontSupportRenameColumn:   true, // 禁用重命名列名
		SkipInitializeWithVersion: true, // 禁用根据当前mysql版本自动配置
	}

	gc := &gorm.Config{
		QueryFields: true, // 根据字段名称查询
		PrepareStmt: true, // 缓存预编译语句
		DryRun:      true, // 生成SQL但不执行
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",   // 数据表名前缀
			SingularTable: true, // 数据表名单数
		},
		NowFunc: func() time.Time {
			return timeUtil.Now() // 当前时间载入时区
		},
		SkipDefaultTransaction:                   true, // 跳过默认事务
		DisableForeignKeyConstraintWhenMigrating: true, // 禁用自动创建外键约束
	}

	gormDB, err := gorm.Open(mysql.New(mc), gc)
	if err != nil {
		panic(err)
	}

	s.SqlDB = sqlDB
	s.SqlMock = sqlMock
	s.GormDB = gormDB
}

// TearDownSuite 结束测试套
func (s *BaseSuite) TearDownSuite() {
	_ = s.SqlDB.Close()
}

// TestStatement 测试表达式
func (s *BaseSuite) TestStatement() {
	stmt := s.GormDB.Statement

	s.T().Log(stmt.SQL.String())
	s.T().Log(stmt.Vars)
	s.T().Log(s.GormDB.Dialector.Explain(stmt.SQL.String(), stmt.Vars...))
}

// TestParseUpdateMap 测试解析更新map
func (s *BaseSuite) TestParseUpdateMap() {
	type UserAuth struct {
		Id     int64  `json:"id" gorm:"primaryKey;autoIncrement;column:id;comment:用户认证id"`                            // 用户认证id
		UserId int64  `json:"user_id" gorm:"column:user_id;type:bigint(20);not null;uniqueIndex:userId;comment:用户id"` // 用户id
		Name   string `json:"name" gorm:"column:name;type:varchar(255);comment:认证名称"`                                 // 认证名称
		Type   int32  `json:"type" gorm:"column:type;type:tinyint(4);default:0;comment:认证类型（0:未认证 1:个人认证 2:企业认证）"`    // 认证类型（0:未认证 1:个人认证 2:企业认证）
		Status int32  `json:"status" gorm:"column:status;type:tinyint(4);default:0;comment:认证状态（0:认证失败 1:认证成功）"`      // 认证状态（0:认证失败 1:认证成功）
		TimeInfo
	}

	m := NewBaseModel(context.Background(), s.GormDB)

	cases := []struct {
		data    interface{}
		omits   []string
		selects []string
		want    map[string]interface{}
	}{
		{
			data:  UserAuth{UserId: 1, Name: "陈测试", Type: 1, Status: 1, TimeInfo: TimeInfo{UpdateTime: 1600000000}},
			omits: nil, selects: nil,
			want: map[string]interface{}{"user_id": int64(1), "name": "陈测试", "type": int32(1), "status": int32(1), "update_time": int64(1600000000)},
		},
		{
			data:  &UserAuth{UserId: 1, Name: "陈测试", Type: 1, Status: 1, TimeInfo: TimeInfo{UpdateTime: 1600000000}},
			omits: []string{"user_id"}, selects: nil,
			want: map[string]interface{}{"name": "陈测试", "type": int32(1), "status": int32(1), "update_time": int64(1600000000)},
		},
		{
			data:  UserAuth{UserId: 1, Name: "陈测试", Type: 0, Status: 0, TimeInfo: TimeInfo{UpdateTime: 1600000000}},
			omits: nil, selects: nil,
			want: map[string]interface{}{"user_id": int64(1), "name": "陈测试", "update_time": int64(1600000000)},
		},
		{
			data:  &UserAuth{UserId: 1, Name: "陈测试", Type: 0, Status: 0, TimeInfo: TimeInfo{UpdateTime: 1600000000}},
			omits: nil, selects: []string{"name", "type", "update_time"},
			want: map[string]interface{}{"name": "陈测试", "type": int32(0), "update_time": int64(1600000000)},
		},
	}

	for _, c := range cases {
		got, err := m.ParseUpdateMap(c.data, c.omits, c.selects)
		s.NoError(err)
		s.Equal(c.want, got)
	}
}
