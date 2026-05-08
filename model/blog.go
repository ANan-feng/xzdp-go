package model

import "time"

// Blog 博客表模型
type Blog struct {
	Id           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ShopId       int64     `gorm:"column:shop_id;not null" json:"shop_id"`
	UserId       int64     `gorm:"column:user_id;not null" json:"user_id"`
	Title        string    `gorm:"column:title;not null" json:"title"`
	Images       string    `gorm:"column:images" json:"images"`
	Content      string    `gorm:"column:content;not null" json:"content"`
	LikedCount   int       `gorm:"column:liked_count;default:0" json:"liked_count"`
	CommentCount int       `gorm:"column:comment_count;default:0" json:"comment_count"`
	CreateTime   time.Time `gorm:"column:create_time;autoCreateTime;<-:create" json:"create_time"`
	UpdateTime   time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	// 非数据库字段
	IsLike bool `gorm:"-" json:"is_like"`
}

// TableName 指定表名
func (b *Blog) TableName() string {
	return "blog"
}
