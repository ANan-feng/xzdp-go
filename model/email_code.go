package model

import "time"

// EmailCode 对应数据库email_code表（实际存储邮箱验证码）
type EmailCode struct {
	Id         int64     `gorm:"column:id" json:"id"`
	Email      string    `gorm:"column:email" json:"email"` // 注释改：邮箱
	Code       string    `gorm:"column:code" json:"code"`
	ExpireTime time.Time `gorm:"column:expire_time" json:"expire_time"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
}

func (s *EmailCode) TableName() string {
	return "email_code"
}
