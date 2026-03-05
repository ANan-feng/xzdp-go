// model/user_dto.go（新增）
package model

// UserDTO 脱敏后的用户信息
type UserDTO struct {
	ID       int64  `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	// 仅保留前端需要的非敏感字段
}

// 转换方法：从User模型转DTO
func (u *User) ToDTO() *UserDTO {
	return &UserDTO{
		ID:       u.Id,
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
	}
}
