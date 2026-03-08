package dao

import (
	"context"
	"xzdp-go/model"
	"xzdp-go/utils"
)

// ShopTypeDAO 商户类型数据访问层
type ShopTypeDAO struct{}

// NewShopTypeDAO 创建商户类型DAO实例
func NewShopTypeDAO() *ShopTypeDAO {
	return &ShopTypeDAO{}
}

// GetById 根据ID查询商户类型
func (d *ShopTypeDAO) GetById(ctx context.Context, typeId int64) (*model.ShopType, error) {
	var shopType model.ShopType
	if err := utils.DB.WithContext(ctx).Where("id = ?", typeId).First(&shopType).Error; err != nil {
		return nil, err
	}
	return &shopType, nil
}

// ListAll 查询所有商户类型（按sort降序）
func (d *ShopTypeDAO) ListAll(ctx context.Context) ([]*model.ShopType, error) {
	var types []*model.ShopType
	if err := utils.DB.WithContext(ctx).Order("sort DESC").Find(&types).Error; err != nil {
		return nil, err
	}
	return types, nil
}

// Create 新增商户类型
func (d *ShopTypeDAO) Create(ctx context.Context, shopType *model.ShopType) error {
	return utils.DB.WithContext(ctx).Create(shopType).Error
}

// Update 更新商户类型
func (d *ShopTypeDAO) Update(ctx context.Context, shopType *model.ShopType) error {
	return utils.DB.WithContext(ctx).Model(&model.ShopType{}).
		Where("id = ?", shopType.ID).
		Select("name", "icon", "sort").
		Updates(map[string]interface{}{
			"name": shopType.Name,
			"icon": shopType.Icon,
			"sort": shopType.Sort,
		}).Error
}

// Delete 根据ID删除商户类型
func (d *ShopTypeDAO) Delete(ctx context.Context, typeId int64) error {
	return utils.DB.WithContext(ctx).Delete(&model.ShopType{}, "id = ?", typeId).Error
}
