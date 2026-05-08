package model

import "time"

// BlogLike 博客点赞表模型
type BlogLike struct {
	Id         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserId     int64     `gorm:"column:user_id;not null" json:"user_id"`
	BlogId     int64     `gorm:"column:blog_id;not null" json:"blog_id"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime;<-:create" json:"create_time"`
}

// TableName 指定表名
func (bl *BlogLike) TableName() string {
	return "blog_like"
}
