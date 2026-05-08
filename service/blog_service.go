package service

import (
	"context"
	"fmt"
	"strconv"
	"time"
	"xzdp-go/dao"
	"xzdp-go/model"
	"xzdp-go/utils"

	"gorm.io/gorm"
)

// BlogService 博客业务逻辑层
type BlogService struct {
	blogDao    *dao.BlogDAO
	commentDao *dao.CommentDAO
	redisDao   *dao.RedisDAO
	userDao    *dao.UserDao
}

func NewBlogService() *BlogService {
	return &BlogService{
		blogDao:    dao.NewBlogDAO(),
		commentDao: dao.NewCommentDAO(),
		redisDao:   dao.NewRedisDAO(),
		userDao:    &dao.UserDao{},
	}
}

// LikeBlog 点赞/取消点赞博客
func (s *BlogService) LikeBlog(ctx context.Context, userId, blogId int64) (bool, error) {
	// 1. 检查Redis缓存是否已点赞
	liked, err := s.redisDao.IsBlogLiked(ctx, blogId, userId)
	if err != nil {
		// Redis失败，查DB
		liked, err = s.blogDao.CheckBlogLiked(ctx, userId, blogId)
		if err != nil {
			return false, fmt.Errorf("检查点赞状态失败: %v", err)
		}
	}

	if liked {
		// 取消点赞
		err = utils.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("user_id = ? AND blog_id = ?", userId, blogId).Delete(&model.BlogLike{}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Blog{}).Where("id = ?", blogId).
				Update("liked_count", gorm.Expr("liked_count - ?", 1)).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return false, fmt.Errorf("取消点赞失败: %v", err)
		}
		// 更新Redis
		s.redisDao.BlogLikeRemove(ctx, blogId, userId)
		// 删除博客详情缓存
		s.redisDao.DelBlogDetailCache(ctx, blogId)
		return false, nil
	}

	// 点赞
	err = utils.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		blogLike := &model.BlogLike{UserId: userId, BlogId: blogId}
		if err := tx.Create(blogLike).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Blog{}).Where("id = ?", blogId).
			Update("liked_count", gorm.Expr("liked_count + ?", 1)).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("点赞失败: %v", err)
	}
	// 更新Redis
	s.redisDao.BlogLikeAdd(ctx, blogId, userId)
	// 删除博客详情缓存
	s.redisDao.DelBlogDetailCache(ctx, blogId)
	return true, nil
}

// GetBlogLikeTopN 获取博客点赞排行榜
func (s *BlogService) GetBlogLikeTopN(ctx context.Context, blogId int64, n int64) ([]map[string]interface{}, error) {
	if n <= 0 {
		n = 10
	}
	// RedisDao 方法只返回用户列表，直接从SortedSet取score需要在此处实现
	key := fmt.Sprintf(utils.BlogLikeKey, blogId)
	entries, err := utils.RedisClient.ZRevRangeWithScores(ctx, key, 0, n-1).Result()
	if err != nil {
		return nil, fmt.Errorf("获取点赞排行榜失败: %v", err)
	}

	result := make([]map[string]interface{}, 0, len(entries))
	for _, z := range entries {
		userId := int64(0)
		switch v := z.Member.(type) {
		case string:
			userId, _ = strconv.ParseInt(v, 10, 64)
		case int64:
			userId = v
		case float64:
			userId = int64(v)
		}
		user, err := s.userDao.GetUserById(ctx, userId)
		if err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"user_id":    user.Id,
			"nickname":   user.Nickname,
			"avatar":     user.Avatar,
			"liked_time": int64(z.Score),
		})
	}
	return result, nil
}

// GetBlogDetail 获取博客详情
func (s *BlogService) GetBlogDetail(ctx context.Context, blogId, userId int64) (*model.Blog, error) {
	// 1. 尝试从缓存获取
	blog, err := s.redisDao.GetBlogDetailCache(ctx, blogId)
	if err == nil && blog != nil {
		// 缓存命中，补充isLike
		liked, _ := s.redisDao.IsBlogLiked(ctx, blogId, userId)
		blog.IsLike = liked
		return blog, nil
	}

	// 2. 缓存miss，从DB查询
	blog, err = s.blogDao.GetBlogById(ctx, blogId)
	if err != nil {
		return nil, fmt.Errorf("获取博客详情失败: %v", err)
	}

	// 3. 补充isLike
	liked, _ := s.redisDao.IsBlogLiked(ctx, blogId, userId)
	blog.IsLike = liked

	// 4. 写入缓存
	s.redisDao.SetBlogDetailCache(ctx, blog, 30*time.Minute)

	return blog, nil
}

// AddComment 添加评论
func (s *BlogService) AddComment(ctx context.Context, userId, blogId, parentId, replyUserId int64, content string) (int64, error) {
	if len(content) < 1 || len(content) > 500 {
		return 0, fmt.Errorf("评论内容长度必须在1-500字符之间")
	}

	// 校验父评论存在且状态正常
	if parentId != 0 {
		parentComment, err := s.commentDao.GetCommentById(ctx, parentId)
		if err != nil {
			return 0, fmt.Errorf("父评论不存在")
		}
		if parentComment.Status != 0 {
			return 0, fmt.Errorf("父评论不可回复")
		}
	}

	comment := &model.BlogComment{
		UserId:      userId,
		BlogId:      blogId,
		ParentId:    parentId,
		ReplyUserId: replyUserId,
		Content:     content,
	}

	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.commentDao.CreateComment(ctx, comment); err != nil {
			return err
		}
		if err := s.blogDao.IncrBlogCommentCount(ctx, blogId); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("添加评论失败: %v", err)
	}

	// 删除评论列表缓存
	s.redisDao.DelCommentListCache(ctx, blogId)

	return comment.Id, nil
}

// DeleteComment 删除评论
func (s *BlogService) DeleteComment(ctx context.Context, userId, commentId int64) error {
	// 获取评论信息
	comment, err := s.commentDao.GetCommentById(ctx, commentId)
	if err != nil {
		return fmt.Errorf("评论不存在: %v", err)
	}

	// 权限校验
	if comment.UserId != userId {
		return fmt.Errorf("无权限删除此评论")
	}

	// 获取子评论数量
	subCount, err := s.commentDao.CountSubComments(ctx, commentId)
	if err != nil {
		return fmt.Errorf("获取子评论数量失败: %v", err)
	}

	totalCount := 1 + subCount

	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.commentDao.DeleteCommentLogical(ctx, commentId); err != nil {
			return err
		}
		if comment.ParentId == 0 {
			// 一级评论，删除子评论
			if err := s.commentDao.DeleteSubCommentsLogical(ctx, commentId); err != nil {
				return err
			}
		}
		if err := s.blogDao.DecrBlogCommentCount(ctx, comment.BlogId, totalCount); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("删除评论失败: %v", err)
	}

	// 删除缓存
	s.redisDao.DelCommentListCache(ctx, comment.BlogId)

	return nil
}

// LikeComment 点赞/取消点赞评论
func (s *BlogService) LikeComment(ctx context.Context, userId, commentId int64) (bool, error) {
	// 检查是否已点赞
	liked, err := s.redisDao.IsCommentLiked(ctx, commentId, userId)
	if err != nil {
		liked, err = s.commentDao.CheckCommentLiked(ctx, userId, commentId)
		if err != nil {
			return false, fmt.Errorf("检查点赞状态失败: %v", err)
		}
	}

	if liked {
		// 取消点赞
		err = utils.DB.Transaction(func(tx *gorm.DB) error {
			if err := s.commentDao.DeleteCommentLike(ctx, userId, commentId); err != nil {
				return err
			}
			if err := s.commentDao.DecrCommentLikedCount(ctx, commentId); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return false, fmt.Errorf("取消点赞失败: %v", err)
		}
		s.redisDao.CommentLikeRemove(ctx, commentId, userId)
		return false, nil
	} else {
		// 点赞
		err = utils.DB.Transaction(func(tx *gorm.DB) error {
			if err := s.commentDao.CreateCommentLike(ctx, userId, commentId); err != nil {
				return err
			}
			if err := s.commentDao.IncrCommentLikedCount(ctx, commentId); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return false, fmt.Errorf("点赞失败: %v", err)
		}
		s.redisDao.CommentLikeAdd(ctx, commentId, userId)
		return true, nil
	}
}

// GetHotComments 获取热门评论列表
func (s *BlogService) GetHotComments(ctx context.Context, blogId, userId int64, page, pageSize int) ([]*model.BlogComment, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 10
	}

	// 尝试从缓存获取
	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1
	commentIds, err := s.redisDao.GetHotCommentIds(ctx, blogId, start, stop)
	if err == nil && len(commentIds) > 0 {
		// 缓存命中，批量获取评论详情（简化）
		comments := make([]*model.BlogComment, 0, len(commentIds))
		for _, idStr := range commentIds {
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				comment, err := s.commentDao.GetCommentById(ctx, id)
				if err == nil {
					comments = append(comments, comment)
				}
			}
		}
		return comments, nil
	}

	// 缓存miss，从DB查询
	comments, err := s.commentDao.GetHotComments(ctx, blogId, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("获取热门评论失败: %v", err)
	}

	// 补充isLike（简化，实际需要扩展结构体）
	for _, comment := range comments {
		_, _ = s.redisDao.IsCommentLiked(ctx, comment.Id, userId)
		// 这里可以添加isLike字段到BlogComment结构体
	}

	// 写入缓存（简化，只缓存ID列表）
	ids := make([]string, len(comments))
	for i, c := range comments {
		ids[i] = fmt.Sprintf("%d", c.Id)
	}
	s.redisDao.SetHotCommentIds(ctx, blogId, ids, 10*time.Minute)

	return comments, nil
}

// GetRecentComments 获取最新评论列表
func (s *BlogService) GetRecentComments(ctx context.Context, blogId, userId int64, page, pageSize int) ([]*model.BlogComment, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 10
	}

	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1
	commentIds, err := s.redisDao.GetRecentCommentIds(ctx, blogId, start, stop)
	if err == nil && len(commentIds) > 0 {
		comments := make([]*model.BlogComment, 0, len(commentIds))
		for _, idStr := range commentIds {
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				comment, err := s.commentDao.GetCommentById(ctx, id)
				if err == nil {
					comments = append(comments, comment)
				}
			}
		}
		return comments, nil
	}

	comments, err := s.commentDao.GetRecentComments(ctx, blogId, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("获取最新评论失败: %v", err)
	}

	for _, comment := range comments {
		comment.IsLike, _ = s.redisDao.IsCommentLiked(ctx, comment.Id, userId)
	}

	ids := make([]string, len(comments))
	for i, c := range comments {
		ids[i] = fmt.Sprintf("%d", c.Id)
	}
	s.redisDao.SetRecentCommentIds(ctx, blogId, ids, 10*time.Minute)

	return comments, nil
}

// GetSubComments 获取子评论列表
func (s *BlogService) GetSubComments(ctx context.Context, parentId, userId int64, page, pageSize int) ([]*model.BlogComment, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 10
	}

	comments, err := s.commentDao.GetSubComments(ctx, parentId, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("获取子评论失败: %v", err)
	}
	for _, comment := range comments {
		comment.IsLike, _ = s.redisDao.IsCommentLiked(ctx, comment.Id, userId)
	}
	return comments, nil
}
