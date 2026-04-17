// model/voucher.go
package model

import "time"

type Voucher struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	ShopID      int64     `gorm:"index;not null"`
	Title       string    `gorm:"not null"`
	SubTitle    string    `gorm:"default:''"`
	Rules       string    `gorm:"default:''"`
	PayValue    int64     `gorm:"not null"`  // 支付金额（分）
	ActualValue int64     `gorm:"not null"`  // 抵扣金额（分）
	Type        int       `gorm:"default:0"` // 0-普通 1-秒杀
	Status      int       `gorm:"default:1"` // 1-上架 0-下架
	CreateTime  time.Time `gorm:"autoCreateTime"`
	UpdateTime  time.Time `gorm:"autoUpdateTime"`
}
