package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"xzdp-go/model"
	"xzdp-go/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SeckillController 秒杀控制器（统一命名规范）
type SeckillController struct{}

// NewSeckillController 创建秒杀控制器实例
func NewSeckillController() *SeckillController {
	return &SeckillController{}
}

// SeckillOrderHandler 秒杀下单接口
// @Summary 秒杀下单
// @Param couponId path int64 true "优惠券ID"
// @Param token header string true "用户Token"
// @Success 200 {object} gin.H{"code":200,"msg":"success","data":{"order_id":int64}}
// @Failure 400 {object} gin.H{"code":400,"msg":"失败原因"}
// @Router /seckill/{couponId} [post]
func (sc *SeckillController) SeckillOrderHandler(c *gin.Context) {
	// 1. 从上下文获取用户ID（auth_middleware已校验登录，直接获取）
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未获取到用户信息"})
		return
	}
	userIdInt64, ok := userId.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "用户ID格式错误"})
		return
	}

	// 2. 解析优惠券ID参数
	couponIdStr := c.Param("couponId")
	var couponId int64
	_, err := fmt.Sscanf(couponIdStr, "%d", &couponId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "优惠券ID格式错误"})
		return
	}

	// 3. 查询优惠券信息（解决表名问题：确保model.Coupon指定正确表名）
	ctx := context.Background()
	coupon := &model.Coupon{}
	if err := utils.DB.WithContext(ctx).Where("id = ?", couponId).First(coupon).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "优惠券不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询优惠券失败：" + err.Error()})
		return
	}

	// 4. 调用Redis预检（Lua脚本）
	result, err := utils.SeckillPreCheck(ctx, couponId, userIdInt64, coupon.EndTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "预检失败：" + err.Error()})
		return
	}

	// 5. 预检结果判断
	switch result {
	case 1:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "优惠券已过期"})
		return
	case 2:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "优惠券库存不足"})
		return
	case 3:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "你已下单过该优惠券（一人一单）"})
		return
	case -1:
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "脚本执行失败"})
		return
	}

	// 6. 数据库层兜底校验 + 生成订单（事务）
	tx := utils.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 6.1 乐观锁扣减库存（解决超卖）
	if err := coupon.UpdateStock(tx); err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "库存不足（兜底校验）"})
		return
	}

	// 6.2 一人一单数据库校验（唯一索引兜底）
	order := &model.SeckillOrder{
		UserID:   userIdInt64,
		CouponID: couponId,
	}
	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		// 唯一索引冲突 → 已下单
		if strings.Contains(err.Error(), "Error 1062: Duplicate entry") {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "你已下单过该优惠券（数据库兜底）"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建订单失败：" + err.Error()})
		return
	}

	// 6.3 提交事务
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "事务提交失败：" + err.Error()})
		return
	}

	// 7. 返回结果
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "秒杀下单成功",
		"data": gin.H{"order_id": order.ID},
	})
}
