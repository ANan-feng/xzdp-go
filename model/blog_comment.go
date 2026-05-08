package model

import "time"

// BlogComment 博客评论表模型
type BlogComment struct {
	Id          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserId      int64     `gorm:"column:user_id;not null" json:"user_id"`
	BlogId      int64     `gorm:"column:blog_id;not null" json:"blog_id"`
	ParentId    int64     `gorm:"column:parent_id;default:0" json:"parent_id"`
	ReplyUserId int64     `gorm:"column:reply_user_id;default:0" json:"reply_user_id"`
	Content     string    `gorm:"column:content;not null" json:"content"`
	LikedCount  int       `gorm:"column:liked_count;default:0" json:"liked_count"`
	Status      int       `gorm:"column:status;default:0" json:"status"`
	CreateTime  time.Time `gorm:"column:create_time;autoCreateTime;<-:create" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	// 非数据库字段
	IsLike bool `gorm:"-" json:"is_like"`
}

// TableName 指定表名
func (bc *BlogComment) TableName() string {
	return "blog_comment"
}
