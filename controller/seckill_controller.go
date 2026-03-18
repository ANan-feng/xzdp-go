package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"xzdp-go/model"
	"xzdp-go/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// 分布式锁相关常量（可抽离到配置文件）
const (
	// 分布式锁key前缀
	seckillLockPrefix = "seckill:lock:"
	// 锁超时时间（防止死锁）
	lockTimeout = 5 * time.Second
	// 锁重试间隔
	lockRetryInterval = 100 * time.Millisecond
	// 最大重试次数
	maxLockRetry = 3
)

// SeckillController 秒杀控制器
type SeckillController struct{}

// NewSeckillController 创建秒杀控制器实例
func NewSeckillController() *SeckillController {
	return &SeckillController{}
}

// ========== 新增：分布式锁工具函数 ==========
// getSeckillLockKey 生成分布式锁key（用户+优惠券维度）
func getSeckillLockKey(userId, couponId int64) string {
	return fmt.Sprintf("%s%d_%d", seckillLockPrefix, userId, couponId)
}

// acquireDistLock 获取分布式锁
// seckill_controller.go 替换分布式锁相关函数
// acquireDistLock 重构：使用RedisLock结构体
func acquireDistLock(ctx context.Context, userId, couponId int64, c *gin.Context) (*utils.RedisLock, bool) {
	lockKey := getSeckillLockKey(userId, couponId)
	// 使用utils的RedisLock
	lock := utils.NewRedisLock(ctx, utils.RedisClient, lockKey, lockTimeout)

	// 重试获取锁
	for i := 0; i < maxLockRetry; i++ {
		success, err := lock.Lock()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取分布式锁失败：" + err.Error()})
			return nil, false
		}
		if success {
			return lock, true
		}
		time.Sleep(lockRetryInterval)
	}

	c.JSON(http.StatusTooManyRequests, gin.H{"code": 429, "msg": "抢购人数过多，请稍后再试"})
	return nil, false
}

// releaseDistLock 重构：关联锁实例
func releaseDistLock(lock *utils.RedisLock) {
	if lock != nil && lock.IsLocked() {
		if err := lock.Unlock(); err != nil {
			fmt.Printf("释放分布式锁失败：err=%v\n", err)
		}
	}
}

// // releaseDistLock 释放分布式锁（Lua脚本保证原子性）
// func releaseDistLock(ctx context.Context, userId, couponId int64, lockValue string) error {
// 	lockKey := getSeckillLockKey(userId, couponId)
// 	// Lua脚本：校验锁标识一致才删除（防止误删其他请求的锁）
// 	luaScript := `
// 		if redis.call('get', KEYS[1]) == ARGV[1] then
// 			return redis.call('del', KEYS[1])
// 		else
// 			return 0
// 		end
// 	`
// 	_, err := utils.RedisEval(ctx, luaScript, []string{lockKey}, []string{lockValue})
// 	return err
// }

// ========== 拆分工具函数：降低主函数复杂度 ==========
// parseCouponId 解析优惠券ID
func parseCouponId(c *gin.Context) (int64, bool) {
	couponIdStr := c.Param("couponId")
	var couponId int64
	_, err := fmt.Sscanf(couponIdStr, "%d", &couponId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "优惠券ID格式错误"})
		return 0, false
	}
	return couponId, true
}

// getSeckillVoucher 获取秒杀优惠券信息
// getSeckillVoucher 函数补充查询店铺ID
func getSeckillVoucher(ctx context.Context, couponId int64, c *gin.Context) (*model.SeckillVoucher, bool) {
	voucher := &model.SeckillVoucher{}
	// 关联查询voucher主表的shop_id
	if err := utils.DB.WithContext(ctx).
		Joins("LEFT JOIN voucher ON seckill_voucher.voucher_id = voucher.id").
		Where("seckill_voucher.voucher_id = ?", couponId).
		Select("seckill_voucher.*, voucher.shop_id").
		First(voucher).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "优惠券不存在"})
			return nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询优惠券失败：" + err.Error()})
		return nil, false
	}
	return voucher, true
}

// checkPreResult 校验Redis预检结果
func checkPreResult(result int, c *gin.Context) bool {
	switch result {
	case 1:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "优惠券已过期"})
		return false
	case 2:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "优惠券库存不足"})
		return false
	case 3:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "你已下单过该优惠券（一人一单）"})
		return false
	case -1:
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "脚本执行失败"})
		return false
	}
	return true
}

// createSeckillOrder 数据库创建秒杀订单（事务）
// 改造createSeckillOrder：移除事务外的锁逻辑，专注数据库操作
func createSeckillOrder(tx *gorm.DB, userId int64, couponId int64, c *gin.Context) (int64, bool) {
	// 1. 扣减库存（乐观锁）
	voucher := &model.SeckillVoucher{VoucherID: couponId}
	if err := tx.Model(voucher).Where("stock > 0").Update("stock", gorm.Expr("stock - ?", 1)).Error; err != nil {
		tx.Rollback()
		// 回滚Redis库存
		rollbackRedisStock(c, couponId)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "扣减库存失败：" + err.Error()})
		return 0, false
	}

	// 2. 检查库存是否扣减成功（乐观锁兜底）
	var count int64
	tx.Model(&model.SeckillVoucher{}).Where("voucher_id = ?", couponId).Select("stock").Scan(&count)
	if count < 0 {
		tx.Rollback()
		// 回滚Redis库存
		rollbackRedisStock(c, couponId)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "库存不足（兜底校验）"})
		return 0, false
	}

	// 3. 创建订单（一人一单：依赖SeckillOrder的user_id+coupon_id唯一索引）
	order := &model.SeckillOrder{
		UserID:     userId,
		VoucherID:  couponId,
		ShopID:     voucher.ShopID,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}
	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		// 回滚Redis库存
		rollbackRedisStock(c, couponId)
		if strings.Contains(err.Error(), "Error 1062: Duplicate entry") {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "你已下单过该优惠券（数据库兜底）"})
			return 0, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建订单失败：" + err.Error()})
		return 0, false
	}

	// 4. 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		rollbackRedisStock(c, couponId)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "事务提交失败：" + err.Error()})
		return 0, false
	}

	return order.ID, true
}

// 新增：Redis库存回滚函数（原子性加1）
func rollbackRedisStock(c *gin.Context, couponId int64) {
	ctx := c.Request.Context()
	stockKey := fmt.Sprintf("xzdp:voucher:stock:%d", couponId)
	// Redis INCR 原子加1，恢复库存
	if err := utils.RedisClient.Incr(ctx, stockKey).Err(); err != nil {
		fmt.Printf("Redis库存回滚失败，couponId=%d，err=%v\n", couponId, err)
	}
}

// SeckillOrderHandler 秒杀下单接口（优化版）
// @Summary 秒杀下单
// @Param couponId path int64 true "优惠券ID"
// @Param token header string true "用户Token"
// @Success 200 {object} gin.H{"code":200,"msg":"success","data":{"order_id":int64}}
// @Failure 400 {object} gin.H{"code":400,"msg":"失败原因"}
// @Failure 429 {object} gin.H{"code":429,"msg":"抢购人数过多，请稍后再试"}
// @Router /seckill/{couponId} [post]
func (sc *SeckillController) SeckillOrderHandler(c *gin.Context) {
	// ========== 1. 基础参数校验 ==========
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未获取到用户信息，请先登录"})
		return
	}
	userIdInt64, ok := userId.(int64)
	if !ok || userIdInt64 <= 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "用户ID格式错误或无效"})
		return
	}

	couponId, ok := parseCouponId(c)
	if !ok || couponId <= 0 {
		return
	}

	// ========== 2. 上下文初始化 ==========
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ========== 3. Redis全量预检（核心拦截） ==========
	voucher, ok := getSeckillVoucher(ctx, couponId, c)
	if !ok {
		return
	}

	// 3.1 Redis原子预检
	result, err := utils.SeckillPreCheck(ctx, couponId, userIdInt64, voucher.EndTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "系统预检失败：" + err.Error()})
		return
	}
	if !checkPreResult(result, c) {
		return
	}

	// ========== 4. 分布式锁（重构后） ==========
	lock, lockSuccess := acquireDistLock(ctx, userIdInt64, couponId, c)
	if !lockSuccess {
		return
	}
	// 确保锁最终释放（无论事务成功/失败）
	defer releaseDistLock(lock)

	// ========== 5. 数据库事务 ==========
	tx := utils.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "开启事务失败：" + tx.Error.Error()})
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			// 回滚Redis库存
			rollbackRedisStock(c, couponId)
			fmt.Printf("[秒杀下单] panic：%v\n", r)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "系统异常，请稍后再试"})
		}
	}()

	// 5.1 创建订单（含库存回滚逻辑）
	orderId, ok := createSeckillOrder(tx, userIdInt64, couponId, c)
	if !ok {
		return
	}

	// ========== 6. 成功响应 ==========
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "秒杀下单成功",
		"data": gin.H{"order_id": orderId},
	})
}

// ========== 自定义类型与结构体 ==========
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

// AddVoucherRequest 添加优惠券请求参数
// seckill_controller.go
type AddVoucherRequest struct {
	ShopID      int64  `json:"shop_id" binding:"required"`            // 商铺ID
	Title       string `json:"title" binding:"required"`              // 标题
	SubTitle    string `json:"sub_title"`                             // 副标题
	Rules       string `json:"rules"`                                 // 使用规则
	PayValue    int64  `json:"pay_value" binding:"required,min=0"`    // 支付金额（分）
	ActualValue int64  `json:"actual_value" binding:"required,min=0"` // 抵扣金额（分）
	// 移除 oneof 规则，改为手动校验
	Type      int        `json:"type" binding:"oneof=0 1"`                // 0-普通 1-秒杀
	Stock     int        `json:"stock" binding:"required_if=Type 1"`      // 库存（秒杀券必填）
	BeginTime CustomTime `json:"begin_time" binding:"required_if=Type 1"` // 开始时间（秒杀券必填）
	EndTime   CustomTime `json:"end_time" binding:"required_if=Type 1"`   // 结束时间（秒杀券必填）
}

// ========== 拆分工具函数：添加优惠券专用 ==========
// validateSeckillVoucher 校验秒杀券参数（提前校验，减少事务内逻辑）
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

// createSeckillVoucher 事务内创建秒杀券（拆分逻辑，降低复杂度）
func createSeckillVoucher(tx *gorm.DB, voucherId int64, req *AddVoucherRequest) error {
	// 插入秒杀券表
	seckillVoucher := &model.SeckillVoucher{
		VoucherID: voucherId,
		Stock:     req.Stock,
		BeginTime: req.BeginTime.ToTime(),
		EndTime:   req.EndTime.ToTime(),
	}
	if err := tx.Create(seckillVoucher).Error; err != nil {
		return fmt.Errorf("创建秒杀券表记录失败：%w", err)
	}

	// 初始化Redis库存
	ctx := context.Background()
	if err := utils.SetCouponStock(ctx, voucherId, int64(req.Stock), req.EndTime.ToTime()); err != nil {
		return fmt.Errorf("初始化Redis库存失败：%w", err)
	}
	return nil
}

// ========== 添加优惠券接口 ==========
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
		errMsg := parseValidationError(err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": errMsg})
		return
	}
	// 临时打印：确认参数绑定结果
	fmt.Printf("绑定后的参数：%+v\n", req) // 查看Type字段是否为0

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

// parseValidationError 解析参数校验错误（拆分工具函数，降低主函数复杂度）
func parseValidationError(err error) string {
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
