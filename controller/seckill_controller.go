package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"xzdp-go/dao"
	"xzdp-go/model"
	"xzdp-go/service"
	"xzdp-go/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type SeckillController struct {
	skService *service.SeckillService
}

func NewSeckillController() *SeckillController {
	return &SeckillController{
		skService: service.NewSeckillService(utils.DB),
	}
}

// SeckillOrderHandler 秒杀下单接口
func (c *SeckillController) SeckillOrderHandler(ctx *gin.Context) {
	// 1. 参数解析
	userID := ctx.GetInt64("userId")
	if userID == 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未登录"})
		return
	}
	voucherIDStr := ctx.Param("couponId")
	voucherID, err := strconv.ParseInt(voucherIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "优惠券ID格式错误"})
		return
	}

	// 2. 调用服务层
	orderID, err := c.skService.CreateSeckillOrder(ctx.Request.Context(), userID, voucherID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// 3. 返回响应
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "抢购成功，订单处理中",
		"data": gin.H{"order_id": orderID},
	})
}

// AddSeckillVoucherHandler 添加秒杀优惠券接口
func (c *SeckillController) AddSeckillVoucherHandler(ctx *gin.Context) {
	// 1. 参数绑定（复用原有结构体）
	var req AddVoucherRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": utils.ParseValidationError(err)})
		return
	}

	// 2. 校验秒杀参数
	if !validateSeckillVoucher(&req, ctx) {
		return
	}

	// 3. 调用服务层初始化库存
	voucherID, err := strconv.ParseInt(ctx.Param("voucherId"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "优惠券ID错误"})
		return
	}
	err = c.skService.InitSeckillStock(ctx.Request.Context(), voucherID, req.Stock, req.EndTime.ToTime())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "初始化秒杀库存失败：" + err.Error()})
		return
	}

	// 4. 返回响应
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "msg": "添加秒杀优惠券成功"})
}

// ========== 优惠券添加接口（无改动，保留完整） ==========
// CustomTime 自定义时间类型（支持 2026-03-03 10:00:00 格式）
type CustomTime time.Time

// UnmarshalJSON 自定义JSON反序列化
func (ct *CustomTime) UnmarshalJSON(data []byte) error {
	timeStr := strings.Trim(string(data), "\"")
	if timeStr == "null" {
		return nil
	}
	format := "2006-01-02 15:04:05"
	t, err := time.Parse(format, timeStr)
	if err != nil {
		return err
	}
	*ct = CustomTime(t)
	return nil
}

// ToTime 将CustomTime转为time.Time
func (ct CustomTime) ToTime() time.Time {
	return time.Time(ct)
}

// ParseValidationError 解析参数校验错误（修正函数名首字母大写，符合导出规则）
func ParseValidationError(err error) string {
	var errMsg string
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, e := range ve {
			// 增加字段值和校验规则的详细信息
			errMsg += fmt.Sprintf("%s字段校验失败：%s（字段值：%v）；", e.Field(), e.Tag(), e.Value())
		}
	} else if strings.Contains(err.Error(), "parsing time") {
		errMsg = "时间格式错误，正确格式：2026-03-03 10:00:00；"
	} else {
		// 打印原始错误（便于调试）
		fmt.Printf("参数绑定原始错误：%v\n", err)
		errMsg = "参数错误：" + err.Error()
	}
	return errMsg
}

// AddVoucher 添加优惠券接口
// @Summary 添加优惠券
// @Description 支持添加普通券和秒杀券（秒杀券需传库存/时间）
// @Tags 优惠券管理
// @Accept json
// @Produce json
// @Param req body AddVoucherRequest true "添加优惠券参数"
// @Success 200 {object} gin.H{"code":0,"msg":"success","data":{"voucher_id":1}}
// @Failure 400 {object} gin.H{"code":400,"msg":"参数错误"}
// @Failure 500 {object} gin.H{"code":500,"msg":"内部错误"}
// @Router /voucher/add [post]
func (sc *SeckillController) AddVoucher(c *gin.Context) {
	// 1. 绑定并校验参数
	var req AddVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 打印原始绑定错误
		fmt.Printf("参数绑定失败：%v\n", err)
		errMsg := ParseValidationError(err) // 修正调用（原utils.ParseValidationError改为本地函数）
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": errMsg})
		return
	}
	// 临时打印：确认参数绑定结果
	fmt.Printf("绑定后的参数：%+v\n", req)

	// 手动校验 Type 的合法性（替代 oneof 规则）
	if req.Type != 0 && req.Type != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "优惠券类型必须是0（普通）或1（秒杀）；"})
		return
	}

	// 2. 提前校验秒杀券参数（事务外校验，减少事务内逻辑）
	if !validateSeckillVoucher(&req, c) {
		return
	}

	// 3. 事务：插入主表 + 秒杀券表（按需）
	var voucherId int64
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		// 3.1 插入优惠券主表
		voucher := &model.Voucher{
			ShopID:      req.ShopID,
			Title:       req.Title,
			SubTitle:    req.SubTitle,
			Rules:       req.Rules,
			PayValue:    req.PayValue,
			ActualValue: req.ActualValue,
			Type:        req.Type,
			Status:      1, // 默认上架
		}
		if err := tx.Create(voucher).Error; err != nil {
			return fmt.Errorf("插入优惠券主表失败：%w", err)
		}
		voucherId = voucher.ID

		// 3.2 秒杀券：插入秒杀券表 + 初始化Redis库存
		if req.Type == 1 {
			if err := createSeckillVoucher(tx, voucherId, &req); err != nil {
				return err
			}
		}
		return nil
	})

	// 4. 处理事务结果
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "添加失败：" + err.Error()})
		return
	}

	// 5. 返回成功响应
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": gin.H{"voucher_id": voucherId}})
}

// createSeckillVoucher 新增：创建秒杀优惠券（原未定义函数）
func createSeckillVoucher(tx *gorm.DB, voucherId int64, req *AddVoucherRequest) error {
	// 插入秒杀券表
	seckillVoucher := &model.SeckillVoucher{
		VoucherID: voucherId,
		Stock:     req.Stock,
		BeginTime: req.BeginTime.ToTime(),
		EndTime:   req.EndTime.ToTime(),
	}
	if err := tx.Create(seckillVoucher).Error; err != nil {
		return fmt.Errorf("插入秒杀券表失败：%w", err)
	}
	// 初始化Redis库存（Cache Aside 写）
	redisDAO := dao.NewRedisDAO()
	if err := redisDAO.SetVoucherStockToCache(context.Background(), voucherId, int64(req.Stock), req.EndTime.ToTime()); err != nil {
		return fmt.Errorf("初始化Redis库存失败：%w", err)
	}
	return nil
}

// QuerySeckillResultHandler 查询秒杀结果（黑马点评标准接口）
func (c *SeckillController) QuerySeckillResultHandler(ctx *gin.Context) {
	// 1. 获取订单ID
	orderIdStr := ctx.Param("orderId")
	orderId, err := strconv.ParseInt(orderIdStr, 10, 64)
	if err != nil {
		fmt.Printf("订单ID解析失败：%v，传入值：%s\n", err, orderIdStr)

		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "订单ID格式错误",
		})
		return
	}

	// 2. 查询订单
	order, err := c.skService.GetSeckillOrderById(ctx.Request.Context(), orderId)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "订单不存在",
		})
		return
	}

	// 3. 返回结果（和黑马点评格式一致）
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "查询成功",
		"data": gin.H{
			"orderId":   order.ID,
			"status":    order.Status,
			"voucherId": order.VoucherID,
		},
	})
}

// /////////////////
// AddVoucherRequest 添加优惠券请求参数
type AddVoucherRequest struct {
	ShopID      int64      `json:"shop_id" binding:"required"`              // 商铺ID
	Title       string     `json:"title" binding:"required"`                // 标题
	SubTitle    string     `json:"sub_title"`                               // 副标题
	Rules       string     `json:"rules"`                                   // 使用规则
	PayValue    int64      `json:"pay_value" binding:"required,min=0"`      // 支付金额（分）
	ActualValue int64      `json:"actual_value" binding:"required,min=0"`   // 抵扣金额（分）
	Type        int        `json:"type" binding:"oneof=0 1"`                // 0-普通 1-秒杀
	Stock       int        `json:"stock" binding:"required_if=Type 1"`      // 库存（秒杀券必填）
	BeginTime   CustomTime `json:"begin_time" binding:"required_if=Type 1"` // 开始时间（秒杀券必填）
	EndTime     CustomTime `json:"end_time" binding:"required_if=Type 1"`   // 结束时间（秒杀券必填）
}

// validateSeckillVoucher 校验秒杀券参数
func validateSeckillVoucher(req *AddVoucherRequest, c *gin.Context) bool {
	if req.Type == 1 {
		// 校验库存
		if req.Stock <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "秒杀券库存必须大于0"})
			return false
		}
		// 校验时间范围
		beginTime := req.BeginTime.ToTime()
		endTime := req.EndTime.ToTime()
		if beginTime.After(endTime) {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "开始时间不能晚于结束时间"})
			return false
		}
		if beginTime.Before(time.Now()) {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "开始时间不能早于当前时间"})
			return false
		}
	}
	return true
}
