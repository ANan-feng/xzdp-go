package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Coupon 优惠券模型
type Coupon struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Title      string    `gorm:"not null" json:"title"`      // 优惠券标题
	Price      float64   `gorm:"not null" json:"price"`      // 优惠券面值
	Stock      int       `gorm:"not null" json:"stock"`      // 库存
	StartTime  time.Time `gorm:"not null" json:"start_time"` // 开始时间
	EndTime    time.Time `gorm:"not null" json:"end_time"`   // 结束时间
	CreateTime time.Time `gorm:"autoCreateTime" json:"-"`
	UpdateTime time.Time `gorm:"autoUpdateTime" json:"-"`
}

// TableName 指定Coupon模型对应的数据库表名（关键！）
func (Coupon) TableName() string {
	return "coupon" // 实际表名是coupon，不是默认的coupons
}

// SeckillOrder 秒杀订单模型
type SeckillOrder struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"not null" json:"user_id"`   // 用户ID
	CouponID   int64     `gorm:"not null" json:"coupon_id"` // 优惠券ID
	CreateTime time.Time `gorm:"autoCreateTime" json:"-"`
	UpdateTime time.Time `gorm:"autoUpdateTime" json:"-"`
}

func (SeckillOrder) TableName() string {
	return "seckill_order"
}

// UpdateStock 乐观锁更新库存（解决超卖）
func (c *Coupon) UpdateStock(tx *gorm.DB) error {
	// 乐观锁：where stock > 0 保证库存不为空，stock = stock -1 原子更新
	result := tx.Model(c).Where("id = ? and stock > 0", c.ID).Update("stock", gorm.Expr("stock - ?", 1))
	if result.Error != nil {
		fmt.Printf("UpdateStock SQL执行失败：%v，SQL：%s\n", result.Error, result.Statement.SQL.String())
		return result.Error
	} else {
		fmt.Printf("UpdateStock 影响行数：%d\n", result.RowsAffected)
	}
	// 影响行数为0说明库存不足
	if result.RowsAffected == 0 {

		return gorm.ErrRecordNotFound
	}
	return nil
}
