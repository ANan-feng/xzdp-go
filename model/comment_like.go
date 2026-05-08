package model

import "time"

// CommentLike 评论点赞表模型
type CommentLike struct {
	Id         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserId     int64     `gorm:"column:user_id;not null" json:"user_id"`
	CommentId  int64     `gorm:"column:comment_id;not null" json:"comment_id"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime;<-:create" json:"create_time"`
}

// TableName 指定表名
func (cl *CommentLike) TableName() string {
	return "comment_like"
}
