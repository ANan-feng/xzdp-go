package model

import "time"

type SeckillOrders struct {
	ID        int64 `gorm:"column:id;primaryKey"`
	UserID    int64 `gorm:"column:user_id"`
	VoucherID int64 `gorm:"column:voucher_id"`
	ShopID    int64 `gorm:"column:shop_id"`
	// 若表中有status字段则保留，无则删除
	Status     int       `gorm:"column:status;default:0"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime"`
}

// 确保表名映射正确
func (SeckillOrders) TableName() string {
	return "seckill_orders"
}
