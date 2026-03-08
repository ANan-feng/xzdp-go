package dao

import (
	"context"
	"xzdp-go/model"
	"xzdp-go/utils"
)

// ShopDAO 商户数据访问层
type ShopDAO struct{}

// NewShopDAO 创建商户DAO实例
func NewShopDAO() *ShopDAO {
	return &ShopDAO{}
}

// GetById 根据ID查询商户（无缓存）
func (d *ShopDAO) GetById(ctx context.Context, shopId int64) (*model.Shop, error) {
	var shop model.Shop
	if err := utils.DB.WithContext(ctx).Where("id = ?", shopId).First(&shop).Error; err != nil {
		return nil, err
	}
	return &shop, nil
}

// Update 更新商户信息（无缓存）
func (d *ShopDAO) Update(ctx context.Context, shop *model.Shop) error {
	return utils.DB.WithContext(ctx).Save(shop).Error
}

// ListByType 根据类型查询商户列表
func (d *ShopDAO) ListByType(ctx context.Context, typeId int64, page, size int) ([]*model.Shop, int64, error) {
	var shops []*model.Shop
	var total int64

	// 统计总数
	if err := utils.DB.Model(&model.Shop{}).Where("type_id = ?", typeId).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * size
	if err := utils.DB.WithContext(ctx).Where("type_id = ?", typeId).
		Offset(offset).Limit(size).Find(&shops).Error; err != nil {
		return nil, 0, err
	}

	return shops, total, nil
}
