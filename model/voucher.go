package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Voucher 优惠券主表模型
type Voucher struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ShopID      int64     `gorm:"not null" json:"shop_id"`                            // 商铺ID
	Title       string    `gorm:"not null" json:"title"`                              // 标题
	SubTitle    string    `gorm:"default:''" json:"sub_title"`                        // 副标题
	Rules       string    `gorm:"default:''" json:"rules"`                            // 使用规则
	PayValue    int64     `gorm:"not null" json:"pay_value"`                          // 支付金额（分）
	ActualValue int64     `gorm:"not null" json:"actual_value"`                       // 抵扣金额（分）
	Type        int       `gorm:"not null;default:0" json:"type" binding:"oneof=0 1"` // 0-普通券 1-秒杀券
	Status      int8      `gorm:"not null;default:1" json:"status"`                   // 1-上架 2-下架 3-过期
	CreateTime  time.Time `gorm:"autoCreateTime" json:"-"`
	UpdateTime  time.Time `gorm:"autoUpdateTime" json:"-"`
}

func (Voucher) TableName() string {
	return "voucher"
}

// SeckillVoucher 秒杀优惠券表（一对一关联Voucher）
type SeckillVoucher struct {
	VoucherID  int64     `gorm:"primaryKey" json:"voucher_id"` // 关联优惠券ID
	Stock      int       `gorm:"not null" json:"stock"`        // 库存
	ShopID     int64     // 新增：关联的店铺ID（非数据库字段，仅用于查询结果接收）
	BeginTime  time.Time `gorm:"not null" json:"begin_time"` // 生效时间
	EndTime    time.Time `gorm:"not null" json:"end_time"`   // 失效时间
	CreateTime time.Time `gorm:"autoCreateTime" json:"-"`
	UpdateTime time.Time `gorm:"autoUpdateTime" json:"-"`
}

func (SeckillVoucher) TableName() string {
	return "seckill_voucher"
}

// SeckillOrder 秒杀订单表（适配新优惠券ID）
type SeckillOrder struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"not null" json:"user_id"`    // 用户ID
	VoucherID  int64     `gorm:"not null" json:"voucher_id"` // 优惠券ID（原CouponID）
	ShopID     int64     `gorm:"not null" json:"shop_id"`    // 商铺ID
	CreateTime time.Time `gorm:"autoCreateTime" json:"-"`
	UpdateTime time.Time `gorm:"autoUpdateTime" json:"-"`
}

func (SeckillOrder) TableName() string {
	return "seckill_order"
}

// UpdateStock 乐观锁更新秒杀券库存（适配新表）
func (s *SeckillVoucher) UpdateStock(tx *gorm.DB) error {
	result := tx.Model(s).Where("voucher_id = ? and stock > 0", s.VoucherID).Update("stock", gorm.Expr("stock - ?", 1))
	if result.Error != nil {
		fmt.Printf("UpdateStock SQL执行失败：%v，SQL：%s\n", result.Error, result.Statement.SQL.String())
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
