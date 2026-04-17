// model/seckill_voucher.go
package model

import "time"

type SeckillVouchers struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	VoucherID  int64     `gorm:"index;not null"` // 关联优惠券ID
	Stock      int       `gorm:"not null"`       // 库存
	BeginTime  time.Time `gorm:"not null"`       // 秒杀开始时间
	EndTime    time.Time `gorm:"not null"`       // 秒杀结束时间
	CreateTime time.Time `gorm:"autoCreateTime"`
	UpdateTime time.Time `gorm:"autoUpdateTime"`
}

// 确保表名映射正确
func (SeckillVouchers) TableName() string {
	return "seckill_vouchers"
}
