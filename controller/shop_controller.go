package controller

import (
	"net/http"
	"strconv"
	"xzdp-go/model"
	"xzdp-go/service"

	"github.com/gin-gonic/gin"
)

// ShopController 商户控制器
type ShopController struct {
	shopService *service.ShopService
}

// NewShopController 创建商户控制器实例
func NewShopController() *ShopController {
	return &ShopController{
		shopService: service.NewShopService(),
	}
}

// GetShopByIdHandler 查询商户详情（通用缓存）
// @Summary 查询商户详情
// @Tags 商户管理
// @Param id path int true "商户ID"
// @Success 200 {object} gin.H{"code":200,"msg":"success","data":model.ShopDTO}
// @Failure 400 {object} gin.H{"code":400,"msg":"参数错误"}
// @Failure 500 {object} gin.H{"code":500,"msg":"服务器内部错误"}
// @Router /shop/{id} [get]
func (c *ShopController) GetShopByIdHandler(ctx *gin.Context) {
	// 1. 解析参数
	shopIdStr := ctx.Param("id")
	shopId, err := strconv.ParseInt(shopIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "商户ID格式错误",
		})
		return
	}

	// 2. 业务逻辑
	shopDTO, err := c.shopService.GetShopById(ctx, shopId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "查询商户失败：" + err.Error(),
		})
		return
	}
	if shopDTO == nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 404,
			"msg":  "商户不存在",
		})
		return
	}

	// 3. 响应
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": shopDTO,
	})
}

// GetHotShopByIdHandler 查询热点商户（逻辑过期缓存）
// @Summary 查询热点商户
// @Tags 商户管理
// @Param id path int true "商户ID"
// @Success 200 {object} gin.H{"code":200,"msg":"success","data":model.ShopDTO}
// @Failure 400 {object} gin.H{"code":400,"msg":"参数错误"}
// @Failure 500 {object} gin.H{"code":500,"msg":"服务器内部错误"}
// @Router /shop/hot/{id} [get]
func (c *ShopController) GetHotShopByIdHandler(ctx *gin.Context) {
	// 1. 解析参数
	shopIdStr := ctx.Param("id")
	shopId, err := strconv.ParseInt(shopIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "商户ID格式错误",
		})
		return
	}

	// 2. 业务逻辑
	shopDTO, err := c.shopService.GetHotShopById(ctx, shopId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "查询热点商户失败：" + err.Error(),
		})
		return
	}
	if shopDTO == nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 404,
			"msg":  "商户不存在",
		})
		return
	}

	// 3. 响应
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": shopDTO,
	})
}

// UpdateShopHandler 更新商户信息
// @Summary 更新商户信息
// @Tags 商户管理
// @Param shop body model.Shop true "商户信息"
// @Success 200 {object} gin.H{"code":200,"msg":"更新成功"}
// @Failure 400 {object} gin.H{"code":400,"msg":"参数错误"}
// @Failure 500 {object} gin.H{"code":500,"msg":"服务器内部错误"}
// @Router /shop [post]
func (c *ShopController) UpdateShopHandler(ctx *gin.Context) {
	// 1. 解析参数
	var shop model.Shop
	if err := ctx.ShouldBindJSON(&shop); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数解析失败：" + err.Error(),
		})
		return
	}
	if shop.ID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "商户ID不能为空",
		})
		return
	}

	// 2. 业务逻辑
	if err := c.shopService.UpdateShop(ctx, &shop); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "更新商户失败：" + err.Error(),
		})
		return
	}

	// 3. 响应
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "更新成功",
	})
}

// ListShopByTypeHandler 按类型分页查询商户
// @Summary 按类型分页查询商户
// @Tags 商户管理
// @Param type_id query int true "商户类型ID"
// @Param page query int true "页码" default(1)
// @Param size query int true "每页条数" default(10)
// @Success 200 {object} gin.H{"code":200,"msg":"success","data":{"list":[]model.ShopDTO,"total":int64}}
// @Failure 400 {object} gin.H{"code":400,"msg":"参数错误"}
// @Failure 500 {object} gin.H{"code":500,"msg":"服务器内部错误"}
// @Router /shop/type [get]
func (c *ShopController) ListShopByTypeHandler(ctx *gin.Context) {
	// 1. 解析参数 - 先判断必填参数是否为空，再做格式转换
	typeIdStr := ctx.Query("type_id")
	// 先校验必填参数是否为空
	if typeIdStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "商户类型ID不能为空",
		})
		return
	}
	// 再转换格式
	typeId, err := strconv.ParseInt(typeIdStr, 10, 64)
	if err != nil || typeId <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "商户类型ID格式错误（必须为正整数）",
		})
		return
	}

	// 处理分页参数（同样优化校验逻辑）
	pageStr := ctx.DefaultQuery("page", "1")
	sizeStr := ctx.DefaultQuery("size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "页码格式错误（必须为正整数）",
		})
		return
	}
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "每页条数格式错误（1-100）",
		})
		return
	}

	// 2. 业务逻辑
	shopList, total, err := c.shopService.ListShopByType(ctx, typeId, page, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "查询商户列表失败：" + err.Error(),
		})
		return
	}

	// 3. 响应
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"list":  shopList,
			"total": total,
		},
	})
}
