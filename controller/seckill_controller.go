package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"xzdp-go/model"
	"xzdp-go/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
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
	// 最大重试次数 1次（快速失败，避免过多等待）
	maxLockRetry = 1
	// Kafka相关常量
	seckillTopic         = "seckill_topic"   // 秒杀消息主题
	msgTimeout           = 5 * time.Second   // 消息发送超时
	asyncResultKeyPrefix = "seckill:result:" // 异步结果存储key前缀
)

// SeckillController 秒杀控制器
type SeckillController struct{}

// NewSeckillController 创建秒杀控制器实例
func NewSeckillController() *SeckillController {
	return &SeckillController{}
}

// ========== 新增：Kafka消息相关函数 ==========
// 构建秒杀消息体（替换为 SeckillRequestMsg）
func buildSeckillMsg(userId, couponId, shopId int64, voucher *model.SeckillVoucher) (*model.SeckillRequestMsg, error) {
	msgID := fmt.Sprintf("seckill_%d_%d_%d", userId, couponId, time.Now().UnixNano())
	msg := &model.SeckillRequestMsg{
		UserID:     userId,
		CouponID:   couponId,
		ShopID:     shopId,
		Stock:      voucher.Stock,
		BeginTime:  voucher.BeginTime,
		EndTime:    voucher.EndTime,
		RequestID:  msgID, // 复用原MsgID为RequestID
		MsgID:      msgID,
		CreateTime: time.Now(),
	}
	return msg, nil
}

// 发送秒杀消息到Kafka
func sendSeckillMsg(ctx context.Context, msg *model.SeckillRequestMsg, c *gin.Context) bool {
	// 序列化消息
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "消息序列化失败：" + err.Error()})
		return false
	}

	// 发送到Kafka（使用 utils.KafkaWriter 替代 KafkaProducer）
	ctx, cancel := context.WithTimeout(ctx, msgTimeout)
	defer cancel()
	if err := utils.KafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.MsgID), // 消息Key（防重复）
		Value: msgBytes,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "发送秒杀消息失败：" + err.Error()})
		return false
	}

	// 记录消息发送日志
	fmt.Printf("秒杀消息发送成功，msgID=%s, userId=%d, couponId=%d\n", msg.MsgID, msg.UserID, msg.CouponID)
	return true
}

// 查询异步秒杀结果（可选：长轮询/短轮询）
func getAsyncSeckillResult(ctx context.Context, msgID string, c *gin.Context) (int64, bool) {
	resultKey := fmt.Sprintf("%s%s", asyncResultKeyPrefix, msgID)
	// 短轮询示例（可改为长轮询）
	for i := 0; i < 10; i++ {
		result, err := utils.RedisClient.Get(ctx, resultKey).Int64()
		if err == nil {
			// 获取结果后删除缓存（避免冗余）
			utils.RedisClient.Del(ctx, resultKey)
			return result, true
		}
		time.Sleep(200 * time.Millisecond)
	}
	c.JSON(http.StatusRequestTimeout, gin.H{"code": 408, "msg": "秒杀请求处理超时，请稍后查询订单状态"})
	return 0, false
}

// ========== 分布式锁工具函数（重构后） ==========
// getSeckillLockKey 生成分布式锁key（用户+优惠券维度）
func getSeckillLockKey(userId, couponId int64) string {
	return fmt.Sprintf("%s%d_%d", seckillLockPrefix, userId, couponId)
}

// acquireDistLock 获取分布式锁
func acquireDistLock(ctx context.Context, userId, couponId int64, c *gin.Context) (*utils.RedisLock, bool) {
	lockKey := getSeckillLockKey(userId, couponId)
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

// releaseDistLock 释放分布式锁
func releaseDistLock(lock *utils.RedisLock) {
	if lock != nil && lock.IsLocked() {
		if err := lock.Unlock(); err != nil {
			fmt.Printf("释放分布式锁失败：err=%v\n", err)
		}
	}
}

// ========== 基础工具函数 ==========
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

// rollbackRedisStock Redis库存回滚
func rollbackRedisStock(c *gin.Context, couponId int64) {
	ctx := c.Request.Context()
	stockKey := fmt.Sprintf("xzdp:voucher:stock:%d", couponId)
	// Redis INCR 原子加1，恢复库存
	if err := utils.RedisClient.Incr(ctx, stockKey).Err(); err != nil {
		fmt.Printf("Redis库存回滚失败，couponId=%d，err=%v\n", couponId, err)
	}
}

// ========== 核心接口：异步秒杀下单 ==========
// SeckillOrderHandler 秒杀下单接口（Kafka异步版）
// @Summary 秒杀下单（异步）
// @Param couponId path int64 true "优惠券ID"
// @Param token header string true "用户Token"
// @Success 200 {object} gin.H{"code":200,"msg":"success","data":{"msg_id":"xxx","order_id":int64}}
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

	// 3.1 Redis原子预检（过滤无效请求）
	result, err := utils.SeckillPreCheckOnly(ctx, couponId, userIdInt64, voucher.EndTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "系统预检失败：" + err.Error()})
		return
	}
	if !checkPreResult(result, c) {
		return
	}

	// ========== 4. 分布式锁（防止重复发送消息） ==========
	lock, lockSuccess := acquireDistLock(ctx, userIdInt64, couponId, c)
	if !lockSuccess {
		return
	}
	defer releaseDistLock(lock)

	// ========== 5. 构建并发送Kafka秒杀消息 ==========
	msg, err := buildSeckillMsg(userIdInt64, couponId, voucher.ShopID, voucher)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "构建秒杀消息失败：" + err.Error()})
		return
	}

	// 发送消息到Kafka
	if !sendSeckillMsg(ctx, msg, c) {
		return
	}

	// ========== 6. 直接返回异步响应（删除同步等待逻辑） ==========
	c.JSON(http.StatusAccepted, gin.H{
		"code": 202,
		"msg":  "秒杀请求已接收，正在排队处理，请稍后查询结果",
		"data": gin.H{
			"msg_id":    msg.MsgID,
			"coupon_id": couponId,
			"query_url": fmt.Sprintf("/seckill/result/%s", msg.MsgID), // 告知用户查询地址
		},
	})
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

// createSeckillVoucher 事务内创建秒杀券
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

// parseValidationError 解析参数校验错误
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

// QuerySeckillResult 查询秒杀结果
// @Summary 查询秒杀结果
// @Param msg_id path string true "消息ID"
// @Router /seckill/result/{msg_id} [get]
func (sc *SeckillController) QuerySeckillResult(c *gin.Context) {
	// 1. 获取消息ID
	msgID := c.Param("msg_id")
	if msgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "消息ID不能为空"})
		return
	}

	// 2. 查询Redis中的秒杀结果
	ctx := context.Background()
	resultKey := fmt.Sprintf("%s%s", asyncResultKeyPrefix, msgID)
	orderID, err := utils.RedisClient.Get(ctx, resultKey).Int64()

	// 3. 处理查询结果
	if err == redis.Nil {
		// 结果未生成（仍在处理中）
		c.JSON(http.StatusOK, gin.H{
			"code": 202,
			"msg":  "秒杀请求正在处理中，请稍后再查",
			"data": gin.H{"msg_id": msgID},
		})
	} else if err != nil {
		// 查询异常
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败：" + err.Error()})
	} else {
		// 查询成功
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"msg":  "秒杀成功",
			"data": gin.H{"order_id": orderID, "msg_id": msgID},
		})
		// 可选：查询后删除结果缓存
		utils.RedisClient.Del(ctx, resultKey)
	}
}
