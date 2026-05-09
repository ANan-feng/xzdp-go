package service

import (
	"context"
	"fmt"
	"time"

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
func (s *UserService) SendEmailCode(ctx context.Context, email string) error {
	if err := utils.ValidateEmailFormat(email); err != nil {
		return err
	}

	if err := utils.CheckEmailSendLimit(ctx, email); err != nil {
		return err
	}

	code := utils.GenerateEmailCode()

	if err := utils.SaveEmailCode(ctx, email, code); err != nil {
		return fmt.Errorf("redis 存储失败: %v", err)
	}

	member := fmt.Sprintf("%d-%d", time.Now().Unix(), time.Now().UnixNano())
	if err := utils.AddEmailSendRecord(ctx, email, member); err != nil {
		_ = utils.DeleteEmailCode(ctx, email)
		return fmt.Errorf("redis 更新发送记录失败: %v", err)
	}

	body := fmt.Sprintf(`<div style="font-family: Arial, sans-serif; padding: 20px; background-color: #f5f5f5;">
		<h2 style="color: #333;">登录验证码</h2>
		<p style="font-size: 16px; color: #666;">您的验证码是：<strong style="color: #007bff; font-size: 20px;">%s</strong></p>
		<p style="font-size: 12px; color: #999;">验证码有效期5分钟，请尽快使用</p>
	</div>`, code)
	if err := utils.SendEmail(email, "登录验证码", body); err != nil {
		_ = utils.DeleteEmailCode(ctx, email)
		_ = utils.RemoveEmailSendRecord(ctx, email, member)
		return fmt.Errorf("发送邮件失败: %v", err)
	}

	return nil
}

// service/user_service.go

// EmailLogin 登录逻辑（重构后）
// 返回值：token, userDTO, error
func (s *UserService) EmailLogin(ctx context.Context, email, code string) (string, *model.UserDTO, error) {
	if email == "" || code == "" {
		return "", nil, fmt.Errorf("邮箱或验证码不能为空")
	}
	// 1. 校验验证码
	if !utils.VerifyAndConsumeCode(ctx, email, code) {
		return "", nil, fmt.Errorf("验证码错误或已过期")
	}

	// 2. 查询用户，不存在则自动创建（自动注册）
	user, err := s.userDao.GetUserByEmail(ctx, email)
	if err != nil {
		// 假设错误是 RecordNotFound，则创建新用户
		user, err = s.userDao.CreateUser(ctx, email)
		if err != nil {
			return "", nil, fmt.Errorf("创建用户失败: %v", err)
		}
	}

	// 3. 生成 Token
	token := utils.GenerateCustomToken()

	// 4. 封装 DTO (脱敏数据)
	userDTO := user.ToDTO()

	// 5. 将 Token 和 UserDTO 存入 Redis
	if err := utils.SetTokenToRedis(ctx, token, user.Id, userDTO); err != nil {
		return "", nil, fmt.Errorf("登录失败，redis异常: %v", err)
	}

	return token, userDTO, nil
}

// GetUserInfo 根据用户ID查询用户信息
func (s *UserService) GetUserInfo(ctx context.Context, userId int64) (*model.User, error) {
	var user model.User
	result := utils.GetDB().WithContext(ctx).First(&user, userId)
	if result.Error != nil {
		return nil, fmt.Errorf("查询用户失败: %v", result.Error)
	}
	return &user, nil
}
