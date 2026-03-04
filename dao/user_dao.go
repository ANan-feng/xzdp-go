package dao

import (
	"errors"
	"time"
	"xzdp-go/model"
	"xzdp-go/utils"
)

// UserDao 用户数据访问层（封装数据库操作，实习：分层解耦）
type UserDao struct{}

// GetUserByEmail 根据手机号查询用户
func (d *UserDao) GetUserByEmail(email string) (*model.User, error) {
	var user model.User
	result := utils.DB.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// CreateUser 创建新用户（短信登录时，手机号不存在则创建）
func (d *UserDao) CreateUser(email string) (*model.User, error) {
	// 1. 校验邮箱非空
	if email == "" {
		return nil, errors.New("邮箱不能为空")
	}

	nickname := "momo"
	now := time.Now()
	user := &model.User{
		Email:      email,
		Nickname:   nickname,                                         // 昵称：黑马用户+手机号后4位
		Avatar:     "https://img-blog.csdnimg.cn/20240101000000.png", // 默认头像
		CreateTime: now,                                              // 显式赋值
		UpdateTime: now,                                              // 显式赋值
	}
	result := utils.DB.Create(user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}
