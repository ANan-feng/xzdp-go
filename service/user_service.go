package service

import (
	"fmt"

	"xzdp-go/dao"
	"xzdp-go/model"
	"xzdp-go/utils"
)

// UserService 用户业务逻辑层（邮箱登录）
type UserService struct {
	userDao *dao.UserDao
}

func NewUserService() *UserService {
	return &UserService{
		userDao: &dao.UserDao{},
	}
}

// SendEmailCode 发送邮箱验证码
func (s *UserService) SendEmailCode(email string) error {
	// 生成6位验证码
	code := utils.GenerateEmailCode()
	// 发送邮箱验证码
	return utils.SendEmailCode(email, code)
}

// EmailLogin 邮箱登录核心逻辑
func (s *UserService) EmailLogin(email, code string) (string, int64, error) {
	// 1. 验证验证码
	if !utils.VerifyEmailCode(email, code) {
		return "", 0, fmt.Errorf("invalid email code")
	}

	// 2. 查询用户（邮箱是否存在）
	user, err := s.userDao.GetUserByEmail(email)
	if err != nil {
		// 3. 邮箱不存在则创建新用户
		user, err = s.userDao.CreateUser(email)
		if err != nil {
			return "", 0, err
		}
	}

	// 4. 生成token
	token := utils.GenerateCustomToken()
	if err != nil {
		return "", 0, err
	}

	return token, user.Id, nil
}

// GetUserInfo 根据用户ID查询用户信息（不变）
func (s *UserService) GetUserInfo(userId int64) (*model.User, error) {
	var user model.User
	result := utils.DB.First(&user, userId)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}
