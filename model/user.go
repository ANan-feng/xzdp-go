package model

import "time"

// User 对应数据库user表（实习注意：字段名与数据库一致，json标签用于接口返回）
type User struct {
	Id         int64     `gorm:"column:id" json:"id"`
	Email      string    `gorm:"column:email" json:"email"`
	Password   string    `gorm:"column:password" json:"-"` // -表示不返回给前端
	Nickname   string    `gorm:"column:nickname" json:"nickname"`
	Avatar     string    `gorm:"column:avatar" json:"avatar"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time" json:"update_time"`
}

// TableName 指定表名（gorm默认复数，这里指定为user）
func (u *User) TableName() string {
	return "user"
}
