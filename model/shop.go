package model

import (
	"time"
)

// Shop 商户表模型
type Shop struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name       string    `gorm:"column:name;not null" json:"name"`
	TypeID     int64     `gorm:"column:type_id;not null" json:"type_id"`
	Images     string    `gorm:"column:images" json:"images"`
	Area       string    `gorm:"column:area" json:"area"`
	Address    string    `gorm:"column:address;not null" json:"address"`
	X          float64   `gorm:"column:x;type:decimal(10,6)" json:"x"`
	Y          float64   `gorm:"column:y;type:decimal(10,6)" json:"y"`
	AvgScore   float64   `gorm:"column:avg_score;type:decimal(2,1)" json:"avg_score"`
	Sold       int       `gorm:"column:sold" json:"sold"`
	Comments   int       `gorm:"column:comments" json:"comments"`
	PriceRange string    `gorm:"column:price_range" json:"price_range"`
	OpenHours  string    `gorm:"column:open_hours" json:"open_hours"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime;<-:create" json:"-"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime" json:"-"`
}

// TableName 指定表名
func (s *Shop) TableName() string {
	return "shop"
}

// ShopDTO 商户数据传输对象（脱敏/简化）
type ShopDTO struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	TypeID    int64   `json:"type_id"`
	Address   string  `json:"address"`
	AvgScore  float64 `json:"avg_score"`
	OpenHours string  `json:"open_hours"`
}
