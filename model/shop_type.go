package model

import (
	"time"
)

// ShopType 商户类型表模型
type ShopType struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name       string    `gorm:"column:name;not null" json:"name"`
	Icon       string    `gorm:"column:icon" json:"icon"`
	Sort       int       `gorm:"column:sort" json:"sort"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime;<-:create" json:"-"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime" json:"-"`
}

// TableName 指定表名
func (st *ShopType) TableName() string {
	return "shop_type"
}

// ShopTypeDTO 商户类型数据传输对象
type ShopTypeDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
	Sort int    `json:"sort"`
}
