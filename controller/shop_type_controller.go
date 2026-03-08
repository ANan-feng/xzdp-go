package controller

import (
	"net/http"
	"strconv"
	"xzdp-go/model"
	"xzdp-go/service"

	"github.com/gin-gonic/gin"
)

// ShopTypeController 商户类型控制器
type ShopTypeController struct {
	shopTypeService *service.ShopTypeService
}

// NewShopTypeController 创建商户类型控制器实例
func NewShopTypeController() *ShopTypeController {
	return &ShopTypeController{
		shopTypeService: service.NewShopTypeService(),
	}
}

// GetShopTypeByIdHandler 根据ID查询商户类型
func (c *ShopTypeController) GetShopTypeByIdHandler(ctx *gin.Context) {
	// 1. 获取参数
	typeIdStr := ctx.Param("id")
	typeId, err := strconv.ParseInt(typeIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的类型ID"})
		return
	}

	// 2. 调用服务
	shopTypeDTO, err := c.shopTypeService.GetShopTypeById(ctx, typeId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败：" + err.Error()})
		return
	}
	if shopTypeDTO == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "商户类型不存在"})
		return
	}

	// 3. 返回结果
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": shopTypeDTO,
	})
}

// ListAllShopTypesHandler 查询所有商户类型
func (c *ShopTypeController) ListAllShopTypesHandler(ctx *gin.Context) {
	// 1. 调用服务
	types, err := c.shopTypeService.ListAllShopTypes(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败：" + err.Error()})
		return
	}

	// 2. 返回结果
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": types,
	})
}

// CreateShopTypeHandler 新增商户类型
func (c *ShopTypeController) CreateShopTypeHandler(ctx *gin.Context) {
	// 1. 绑定参数
	var shopType model.ShopType
	if err := ctx.ShouldBindJSON(&shopType); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：" + err.Error()})
		return
	}

	// 2. 调用服务
	if err := c.shopTypeService.CreateShopType(ctx, &shopType); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败：" + err.Error()})
		return
	}

	// 3. 返回结果
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "创建成功",
	})
}

// UpdateShopTypeHandler 更新商户类型
func (c *ShopTypeController) UpdateShopTypeHandler(ctx *gin.Context) {
	// 1. 绑定参数
	var shopType model.ShopType
	if err := ctx.ShouldBindJSON(&shopType); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：" + err.Error()})
		return
	}
	if shopType.ID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "类型ID不能为空"})
		return
	}

	// 2. 调用服务
	if err := c.shopTypeService.UpdateShopType(ctx, &shopType); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败：" + err.Error()})
		return
	}

	// 3. 返回结果
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "更新成功",
	})
}

// DeleteShopTypeHandler 删除商户类型
func (c *ShopTypeController) DeleteShopTypeHandler(ctx *gin.Context) {
	// 1. 获取参数
	typeIdStr := ctx.Param("id")
	typeId, err := strconv.ParseInt(typeIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的类型ID"})
		return
	}

	// 2. 调用服务
	if err := c.shopTypeService.DeleteShopType(ctx, typeId); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败：" + err.Error()})
		return
	}

	// 3. 返回结果
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "删除成功",
	})
}
