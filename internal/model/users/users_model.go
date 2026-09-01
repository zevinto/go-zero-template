// Package users 提供 User 的 gorm 数据访问。
// model 持有 *gorm.DB，用 gorm 链式 API 做 CRUD 查询；logic 层经 ServiceContext 使用。
package users

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrNotFound 未找到记录时返回（gorm.ErrRecordNotFound 别名）。
var ErrNotFound = gorm.ErrRecordNotFound

// User 对应 users 表；表名由 TableName 显式指定，避免 gorm 复数化差异。
type User struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	Name      string `gorm:"column:name"`
	Email     string `gorm:"column:email"`
	CreatedAt int64  `gorm:"column:created_at"`
	UpdatedAt int64  `gorm:"column:updated_at"`
}

// TableName 明确指定表名。
func (User) TableName() string { return "users" }

// Users gorm 数据访问接口，logic 层依赖此接口而不直接碰 *gorm.DB。
type Users interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, id int64, u *User) error
	Delete(ctx context.Context, id int64) error
}

// UsersModel 持 gorm.DB 的默认实现。
type UsersModel struct{ db *gorm.DB }

// NewUsersModel 基于 gorm 连接构造 Users。
func NewUsersModel(db *gorm.DB) *UsersModel { return &UsersModel{db: db} }

func (m *UsersModel) Create(ctx context.Context, u *User) error {
	if err := m.db.WithContext(ctx).Create(u).Error; err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}
	return nil
}

func (m *UsersModel) GetByID(ctx context.Context, id int64) (*User, error) {
	var u User
	if err := m.db.WithContext(ctx).Where("id = ?", id).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (m *UsersModel) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	if err := m.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (m *UsersModel) Update(ctx context.Context, id int64, u *User) error {
	res := m.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).
		Updates(map[string]any{
			"name":       u.Name,
			"email":      u.Email,
			"updated_at": u.UpdatedAt,
		})
	if res.Error != nil {
		return fmt.Errorf("更新用户失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *UsersModel) Delete(ctx context.Context, id int64) error {
	res := m.db.WithContext(ctx).Where("id = ?", id).Delete(&User{})
	if res.Error != nil {
		return fmt.Errorf("删除用户失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
