package controller

import (
	"net/http"
	"strconv"
	"xzdp-go/service"

	"github.com/gin-gonic/gin"
)

type BlogController struct {
	blogService *service.BlogService
}

func NewBlogController() *BlogController {
	return &BlogController{
		blogService: service.NewBlogService(),
	}
}

// GetBlogDetail 获取博客详情
func (c *BlogController) GetBlogDetail(ctx *gin.Context) {
	blogIdStr := ctx.Param("id")
	blogId, err := strconv.ParseInt(blogIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "博客ID格式错误",
		})
		return
	}

	userId := ctx.GetInt64("userId")
	blog, err := c.blogService.GetBlogDetail(ctx.Request.Context(), blogId, userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": blog,
	})
}

// LikeBlog 点赞/取消点赞博客
func (c *BlogController) LikeBlog(ctx *gin.Context) {
	blogIdStr := ctx.Param("id")
	blogId, err := strconv.ParseInt(blogIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "博客ID格式错误",
		})
		return
	}

	userId := ctx.GetInt64("userId")
	liked, err := c.blogService.LikeBlog(ctx.Request.Context(), userId, blogId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"liked": liked,
		},
	})
}

// GetBlogLikeTopN 获取博客点赞排行榜
func (c *BlogController) GetBlogLikeTopN(ctx *gin.Context) {
	blogIdStr := ctx.Param("id")
	blogId, err := strconv.ParseInt(blogIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "博客ID格式错误",
		})
		return
	}

	nStr := ctx.DefaultQuery("n", "10")
	n, err := strconv.ParseInt(nStr, 10, 64)
	if err != nil {
		n = 10
	}

	topN, err := c.blogService.GetBlogLikeTopN(ctx.Request.Context(), blogId, n)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": topN,
	})
}

// GetHotComments 获取热门评论列表
func (c *BlogController) GetHotComments(ctx *gin.Context) {
	blogIdStr := ctx.Param("id")
	blogId, err := strconv.ParseInt(blogIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "博客ID格式错误",
		})
		return
	}

	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("pageSize", "10")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	userId := ctx.GetInt64("userId")
	comments, err := c.blogService.GetHotComments(ctx.Request.Context(), blogId, userId, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": comments,
	})
}

// GetRecentComments 获取最新评论列表
func (c *BlogController) GetRecentComments(ctx *gin.Context) {
	blogIdStr := ctx.Param("id")
	blogId, err := strconv.ParseInt(blogIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "博客ID格式错误",
		})
		return
	}

	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("pageSize", "10")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	userId := ctx.GetInt64("userId")
	comments, err := c.blogService.GetRecentComments(ctx.Request.Context(), blogId, userId, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": comments,
	})
}

// AddComment 添加评论
func (c *BlogController) AddComment(ctx *gin.Context) {
	blogIdStr := ctx.Param("id")
	blogId, err := strconv.ParseInt(blogIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "博客ID格式错误",
		})
		return
	}

	var req struct {
		Content     string `json:"content"`
		ParentId    int64  `json:"parentId"`
		ReplyUserId int64  `json:"replyUserId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "请求参数错误",
		})
		return
	}

	userId := ctx.GetInt64("userId")
	commentId, err := c.blogService.AddComment(ctx.Request.Context(), userId, blogId, req.ParentId, req.ReplyUserId, req.Content)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"id": commentId,
		},
	})
}

// DeleteComment 删除评论
func (c *BlogController) DeleteComment(ctx *gin.Context) {
	commentIdStr := ctx.Param("id")
	commentId, err := strconv.ParseInt(commentIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "评论ID格式错误",
		})
		return
	}

	userId := ctx.GetInt64("userId")
	err = c.blogService.DeleteComment(ctx.Request.Context(), userId, commentId)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
	})
}

// LikeComment 点赞/取消点赞评论
func (c *BlogController) LikeComment(ctx *gin.Context) {
	commentIdStr := ctx.Param("id")
	commentId, err := strconv.ParseInt(commentIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "评论ID格式错误",
		})
		return
	}

	userId := ctx.GetInt64("userId")
	liked, err := c.blogService.LikeComment(ctx.Request.Context(), userId, commentId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"liked": liked,
		},
	})
}

// GetSubComments 获取子评论列表
func (c *BlogController) GetSubComments(ctx *gin.Context) {
	commentIdStr := ctx.Param("id")
	commentId, err := strconv.ParseInt(commentIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "评论ID格式错误",
		})
		return
	}

	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("pageSize", "10")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	userId := ctx.GetInt64("userId")
	comments, err := c.blogService.GetSubComments(ctx.Request.Context(), commentId, userId, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": comments,
	})
}
