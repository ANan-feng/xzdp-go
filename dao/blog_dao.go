package dao

import (
	"context"
	"xzdp-go/model"
	"xzdp-go/utils"

	"gorm.io/gorm"
)

// BlogDAO 博客数据访问层
type BlogDAO struct{}

// NewBlogDAO 创建博客DAO实例
func NewBlogDAO() *BlogDAO {
	return &BlogDAO{}
}

// GetBlogById 根据ID查询博客
func (d *BlogDAO) GetBlogById(ctx context.Context, blogId int64) (*model.Blog, error) {
	var blog model.Blog
	if err := utils.DB.WithContext(ctx).Where("id = ?", blogId).First(&blog).Error; err != nil {
		return nil, err
	}
	return &blog, nil
}

// GetBlogsByShopId 根据店铺ID查询博客列表
func (d *BlogDAO) GetBlogsByShopId(ctx context.Context, shopId int64, page, pageSize int) ([]*model.Blog, error) {
	var blogs []*model.Blog
	offset := (page - 1) * pageSize
	if err := utils.DB.WithContext(ctx).Where("shop_id = ?", shopId).
		Offset(offset).Limit(pageSize).Find(&blogs).Error; err != nil {
		return nil, err
	}
	return blogs, nil
}

// CreateBlogLike 创建博客点赞
func (d *BlogDAO) CreateBlogLike(ctx context.Context, userId, blogId int64) error {
	blogLike := &model.BlogLike{
		UserId: userId,
		BlogId: blogId,
	}
	return utils.DB.WithContext(ctx).Create(blogLike).Error
}

// DeleteBlogLike 删除博客点赞
func (d *BlogDAO) DeleteBlogLike(ctx context.Context, userId, blogId int64) error {
	return utils.DB.WithContext(ctx).Where("user_id = ? AND blog_id = ?", userId, blogId).
		Delete(&model.BlogLike{}).Error
}

// CheckBlogLiked 检查是否已点赞博客
func (d *BlogDAO) CheckBlogLiked(ctx context.Context, userId, blogId int64) (bool, error) {
	var count int64
	err := utils.DB.WithContext(ctx).Model(&model.BlogLike{}).
		Where("user_id = ? AND blog_id = ?", userId, blogId).Count(&count).Error
	return count > 0, err
}

// IncrBlogLikedCount 增加博客点赞数
func (d *BlogDAO) IncrBlogLikedCount(ctx context.Context, blogId int64) error {
	return utils.DB.WithContext(ctx).Model(&model.Blog{}).
		Where("id = ?", blogId).Update("liked_count", gorm.Expr("liked_count + ?", 1)).Error
}

// DecrBlogLikedCount 减少博客点赞数
func (d *BlogDAO) DecrBlogLikedCount(ctx context.Context, blogId int64) error {
	return utils.DB.WithContext(ctx).Model(&model.Blog{}).
		Where("id = ?", blogId).Update("liked_count", gorm.Expr("liked_count - ?", 1)).Error
}

// IncrBlogCommentCount 增加博客评论数
func (d *BlogDAO) IncrBlogCommentCount(ctx context.Context, blogId int64) error {
	return utils.DB.WithContext(ctx).Model(&model.Blog{}).
		Where("id = ?", blogId).Update("comment_count", gorm.Expr("comment_count + ?", 1)).Error
}

// DecrBlogCommentCount 减少博客评论数
func (d *BlogDAO) DecrBlogCommentCount(ctx context.Context, blogId, count int64) error {
	return utils.DB.WithContext(ctx).Model(&model.Blog{}).
		Where("id = ?", blogId).Update("comment_count", gorm.Expr("comment_count - ?", count)).Error
}
