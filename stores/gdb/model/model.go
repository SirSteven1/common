package model

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"cxqi/common/errcode"
	"cxqi/common/jwt"
	sliceUtil "cxqi/common/kit/slice"
	logging "cxqi/common/logger"
	"cxqi/common/logger/xzap"
)

// ContextKey 上下文key类型
type ContextKey string

// String 实现序列化字符串方法
func (c ContextKey) String() string {
	return "model context key: " + string(c)
}

// BaseModel 基础模型
type BaseModel struct {
	Ctx context.Context
	DB  *gorm.DB
	logging.Logger
}

// NewBaseModel 新建基础模型
func NewBaseModel(ctx context.Context, db *gorm.DB) *BaseModel {
	return &BaseModel{
		Ctx:    ctx,
		DB:     db.WithContext(ctx),
		Logger: xzap.WithContext(ctx),
	}
}

// ParseUpdateMap 解析模型数据获取对应的更新map，更新时间字段会自动跟踪，参数
//   data 须为模型数据结构体或其所对应指针
//   omits 不更新的字段列表，只在selects长度为0时生效
//   selects 指定更新的字段列表，长度为0时，生成的更新map取模型数据非零值并忽略主键和创建时间字段
//   notIgnores 不忽略的字段列表，只在selects长度为0时生效，与omits冲突时，以omits为准
func (m *BaseModel) ParseUpdateMap(data interface{}, omits, selects []string, notIgnores ...string) (map[string]interface{}, error) {
	db := m.DB.WithContext(m.Ctx)
	stmt := db.Statement
	err := stmt.Parse(data)
	if err != nil {
		return nil, err
	}

	rv := reflect.ValueOf(data)
	um := make(map[string]interface{})

	if len(selects) > 0 {
		sm := sliceUtil.CountString(selects)
		for _, f := range stmt.Schema.Fields {
			if _, ok := sm[f.DBName]; ok {
				fv, _ := f.ValueOf(rv)
				um[f.DBName] = fv
			} else if f.AutoUpdateTime > 0 {
				// 跟踪更新时间字段
				um[f.DBName] = GetTimestamp(db.NowFunc(), f.AutoUpdateTime)
			}
		}
	} else {
		om := sliceUtil.CountString(omits)
		nim := sliceUtil.CountString(notIgnores)
		for _, f := range stmt.Schema.Fields {
			if _, ok := om[f.DBName]; !ok {
				fv, zero := f.ValueOf(rv)
				if _, ok := nim[f.DBName]; ok {
					um[f.DBName] = fv
				} else if !f.PrimaryKey && f.AutoCreateTime == 0 && !zero {
					// 不为主键、创建时间字段和零值
					um[f.DBName] = fv
				} else if f.AutoUpdateTime > 0 {
					// 跟踪更新时间字段
					um[f.DBName] = GetTimestamp(db.NowFunc(), f.AutoUpdateTime)
				}
			}
		}
	}

	return um, nil
}

// Update 通过id更新数据信息
func (m *BaseModel) Update(data IdGetter, selects []string) error {
	if data.GetId() < 1 {
		return errcode.ErrInvalidParams
	}

	db := m.DB.Model(data)
	if len(selects) > 0 {
		return db.Select(selects).Updates(data).Error
	}

	return db.Updates(data).Error
}

// UpdateByCondition 通过动态条件更新数据信息
func (m *BaseModel) UpdateByCondition(data schema.Tabler, condition func(*gorm.DB) *gorm.DB) error {
	return m.DB.Model(data).Scopes(condition).Updates(data).Error
}

// UpdateWithMapByCondition 使用map通过动态条件更新数据信息
func (m *BaseModel) UpdateWithMapByCondition(t schema.Tabler, data map[string]interface{}, condition func(*gorm.DB) *gorm.DB) error {
	return m.DB.Model(t).Scopes(condition).Updates(data).Error
}

// CountByCondition 通过动态条件计数数据信息
func (m *BaseModel) CountByCondition(data schema.Tabler, condition func(*gorm.DB) *gorm.DB) (int64, error) {
	var count int64
	err := m.DB.Model(data).Scopes(condition).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// PluckInt64s 通过动态条件查找单个列并将结果扫描到int64切片中
func (m *BaseModel) PluckInt64s(data schema.Tabler, column string, condition func(*gorm.DB) *gorm.DB) ([]int64, error) {
	var s []int64
	err := m.DB.Model(data).Scopes(condition).Pluck(column, &s).Error
	if err != nil {
		return nil, err
	}

	return s, nil
}

// PluckStrings 通过动态条件查找单个列并将结果扫描到string切片中
func (m *BaseModel) PluckStrings(data schema.Tabler, column string, condition func(*gorm.DB) *gorm.DB) ([]string, error) {
	var s []string
	err := m.DB.Model(data).Scopes(condition).Pluck(column, &s).Error
	if err != nil {
		return nil, err
	}

	return s, nil
}

// DeleteByCondition 通过动态条件删除数据信息
func (m *BaseModel) DeleteByCondition(data schema.Tabler, condition func(*gorm.DB) *gorm.DB) error {
	return m.DB.Scopes(condition).Delete(data).Error
}

// IdGetter id获取器
type IdGetter interface {
	schema.Tabler
	// GetId 获取id
	GetId() int64
}

// TimeInfo 通用时间信息
type TimeInfo struct {
	CreateTime int64 `json:"create_time" gorm:"column:create_time;type:bigint(20);autoCreateTime:milli;comment:创建时间"` // 创建时间
	UpdateTime int64 `json:"update_time" gorm:"column:update_time;type:bigint(20);autoUpdateTime:milli;comment:更新时间"` // 更新时间
}

// ByInfo 操作者信息
type ByInfo struct {
	CreateBy int64 `json:"create_by" gorm:"column:create_by;type:bigint(20);comment:创建者"` // 创建者
	UpdateBy int64 `json:"update_by" gorm:"column:update_by;type:bigint(20);comment:更新者"` // 更新者
}

// BeforeCreate 创建前钩子函数
func (bi *ByInfo) BeforeCreate(tx *gorm.DB) error {
	if token, ok := jwt.FromContext(tx.Statement.Context); ok {
		if bi.CreateBy < 1 {
			bi.CreateBy = token.UserId
		}
		if bi.UpdateBy < 1 {
			bi.UpdateBy = token.UserId
		}
	}

	return nil
}

// BeforeUpdate 更新前钩子函数
func (bi *ByInfo) BeforeUpdate(tx *gorm.DB) error {
	if token, ok := jwt.FromContext(tx.Statement.Context); ok {
		if bi.UpdateBy < 1 {
			bi.UpdateBy = token.UserId
		}
	}

	return nil
}

// PaginationQuery 分页查询
type PaginationQuery struct {
	page     int                       // 页数
	pageSize int                       // 每页大小
	order    string                    // 排序
	queries  []func(*gorm.DB) *gorm.DB // 查询条件
}

// NewPaginationQuery 新建分页查询
func NewPaginationQuery(page, pageSize int64, order string) *PaginationQuery {
	return &PaginationQuery{
		page:     int(page),
		pageSize: int(pageSize),
		order:    order,
	}
}

// Add 添加查询条件
func (pq *PaginationQuery) Add(query func(*gorm.DB) *gorm.DB) {
	pq.queries = append(pq.queries, query)
}

// Queries 获取查询条件
func (pq *PaginationQuery) Queries() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Scopes(pq.queries...)
	}
}

// Paginate 构建分页查询
func (pq *PaginationQuery) Paginate() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		limit := pq.pageSize
		offset := pq.pageSize * (pq.page - 1)

		if pq.order != "" {
			return db.Scopes(pq.queries...).Order(pq.order).Limit(limit).Offset(offset)
		}
		return db.Scopes(pq.queries...).Limit(limit).Offset(offset)
	}
}

// IsRecordNotFound 判断记录是否被找到
func IsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// GetTimestamp 获取时间类型对应的当前时间戳
func GetTimestamp(t time.Time, tt schema.TimeType) (ts int64) {
	switch tt {
	case schema.UnixMillisecond:
		ts = t.UnixNano() / int64(time.Millisecond)
	case schema.UnixNanosecond:
		ts = t.UnixNano()
	default:
		ts = t.UnixNano() / int64(time.Millisecond)
	}

	return
}

// FmtFields 格式化数据字段
func FmtFields(tableName string, fields ...string) []string {
	var fmtFields []string
	for _, field := range fields {
		fmtFields = append(fmtFields, fmt.Sprintf("`%s`.`%s`", tableName, field))
	}
	return fmtFields
}
