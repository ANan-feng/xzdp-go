package dao

import (
	"context"
	"xzdp-go/model"
	"xzdp-go/utils"

	"gorm.io/gorm"
)

// CommentDAO 评论数据访问层
type CommentDAO struct{}

// NewCommentDAO 创建评论DAO实例
func NewCommentDAO() *CommentDAO {
	return &CommentDAO{}
}

// CreateComment 创建评论
func (d *CommentDAO) CreateComment(ctx context.Context, comment *model.BlogComment) error {
	return utils.DB.WithContext(ctx).Create(comment).Error
}

// DeleteCommentLogical 逻辑删除评论
func (d *CommentDAO) DeleteCommentLogical(ctx context.Context, commentId int64) error {
	return utils.DB.WithContext(ctx).Model(&model.BlogComment{}).
		Where("id = ?", commentId).Update("status", 2).Error
}

// DeleteSubCommentsLogical 逻辑删除子评论
func (d *CommentDAO) DeleteSubCommentsLogical(ctx context.Context, parentId int64) error {
	return utils.DB.WithContext(ctx).Model(&model.BlogComment{}).
		Where("parent_id = ?", parentId).Update("status", 2).Error
}

// GetCommentById 根据ID查询评论
func (d *CommentDAO) GetCommentById(ctx context.Context, commentId int64) (*model.BlogComment, error) {
	var comment model.BlogComment
	if err := utils.DB.WithContext(ctx).Where("id = ?", commentId).First(&comment).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

// GetHotComments 获取热门评论列表
func (d *CommentDAO) GetHotComments(ctx context.Context, blogId int64, page, pageSize int) ([]*model.BlogComment, error) {
	var comments []*model.BlogComment
	offset := (page - 1) * pageSize
	if err := utils.DB.WithContext(ctx).Where("blog_id = ? AND parent_id = 0 AND status = 0", blogId).
		Order("liked_count DESC, create_time DESC").
		Offset(offset).Limit(pageSize).Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

// GetRecentComments 获取最新评论列表
func (d *CommentDAO) GetRecentComments(ctx context.Context, blogId int64, page, pageSize int) ([]*model.BlogComment, error) {
	var comments []*model.BlogComment
	offset := (page - 1) * pageSize
	if err := utils.DB.WithContext(ctx).Where("blog_id = ? AND parent_id = 0 AND status = 0", blogId).
		Order("create_time DESC").
		Offset(offset).Limit(pageSize).Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

// GetSubComments 获取子评论列表
func (d *CommentDAO) GetSubComments(ctx context.Context, parentId int64, page, pageSize int) ([]*model.BlogComment, error) {
	var comments []*model.BlogComment
	offset := (page - 1) * pageSize
	if err := utils.DB.WithContext(ctx).Where("parent_id = ? AND status = 0", parentId).
		Order("create_time ASC").
		Offset(offset).Limit(pageSize).Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

// CountSubComments 统计子评论数量
func (d *CommentDAO) CountSubComments(ctx context.Context, parentId int64) (int64, error) {
	var count int64
	err := utils.DB.WithContext(ctx).Model(&model.BlogComment{}).
		Where("parent_id = ? AND status = 0", parentId).Count(&count).Error
	return count, err
}

// CreateCommentLike 创建评论点赞
func (d *CommentDAO) CreateCommentLike(ctx context.Context, userId, commentId int64) error {
	commentLike := &model.CommentLike{
		UserId:    userId,
		CommentId: commentId,
	}
	return utils.DB.WithContext(ctx).Create(commentLike).Error
}

// DeleteCommentLike 删除评论点赞
func (d *CommentDAO) DeleteCommentLike(ctx context.Context, userId, commentId int64) error {
	return utils.DB.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userId, commentId).
		Delete(&model.CommentLike{}).Error
}

// CheckCommentLiked 检查是否已点赞评论
func (d *CommentDAO) CheckCommentLiked(ctx context.Context, userId, commentId int64) (bool, error) {
	var count int64
	err := utils.DB.WithContext(ctx).Model(&model.CommentLike{}).
		Where("user_id = ? AND comment_id = ?", userId, commentId).Count(&count).Error
	return count > 0, err
}

// IncrCommentLikedCount 增加评论点赞数
func (d *CommentDAO) IncrCommentLikedCount(ctx context.Context, commentId int64) error {
	return utils.DB.WithContext(ctx).Model(&model.BlogComment{}).
		Where("id = ?", commentId).Update("liked_count", gorm.Expr("liked_count + ?", 1)).Error
}

// DecrCommentLikedCount 减少评论点赞数
func (d *CommentDAO) DecrCommentLikedCount(ctx context.Context, commentId int64) error {
	return utils.DB.WithContext(ctx).Model(&model.BlogComment{}).
		Where("id = ?", commentId).Update("liked_count", gorm.Expr("liked_count - ?", 1)).Error
}
