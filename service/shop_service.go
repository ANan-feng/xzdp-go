package service

import (
	"context"
	"xzdp-go/dao"
	"xzdp-go/model"
	"xzdp-go/utils"
)

// ShopService 商户业务逻辑层
type ShopService struct {
	shopDAO *dao.ShopDAO
}

// NewShopService 创建商户服务实例
func NewShopService() *ShopService {
	return &ShopService{
		shopDAO: dao.NewShopDAO(),
	}
}

// GetShopById 通用查询（带缓存穿透+雪崩防护）
func (s *ShopService) GetShopById(ctx context.Context, shopId int64) (*model.ShopDTO, error) {
	// 1. 从缓存+数据库查询
	shop, err := utils.GetShopByIdWithCache(ctx, shopId)
	if err != nil {
		return nil, err
	}
	if shop == nil {
		return nil, nil
	}

	// 2. 转换为DTO（脱敏/简化）
	dto := &model.ShopDTO{
		ID:        shop.ID,
		Name:      shop.Name,
		TypeID:    shop.TypeID,
		Address:   shop.Address,
		AvgScore:  shop.AvgScore,
		OpenHours: shop.OpenHours,
	}

	return dto, nil
}

// GetHotShopById 热点商户查询（逻辑过期方案）
func (s *ShopService) GetHotShopById(ctx context.Context, shopId int64) (*model.ShopDTO, error) {
	// 1. 逻辑过期缓存查询
	shop, err := utils.GetShopByIdWithLogicalExpire(ctx, shopId)
	if err != nil {
		return nil, err
	}
	if shop == nil {
		return nil, nil
	}

	// 2. 转换为DTO
	dto := &model.ShopDTO{
		ID:        shop.ID,
		Name:      shop.Name,
		TypeID:    shop.TypeID,
		Address:   shop.Address,
		AvgScore:  shop.AvgScore,
		OpenHours: shop.OpenHours,
	}

	return dto, nil
}

// UpdateShop 更新商户（Cache Aside模式）
func (s *ShopService) UpdateShop(ctx context.Context, shop *model.Shop) error {
	// 改用 GORM 的 Select 方法，只更新需要的字段，避免更新 create_time
	err := utils.DB.WithContext(ctx).Model(&model.Shop{}).
		Where("id = ?", shop.ID).
		Select("name", "type_id", "images", "area", "address", "x", "y", "avg_score", "sold", "comments", "price_range", "open_hours").
		Updates(map[string]interface{}{
			"name":        shop.Name,
			"type_id":     shop.TypeID,
			"images":      shop.Images,
			"area":        shop.Area,
			"address":     shop.Address,
			"x":           shop.X,
			"y":           shop.Y,
			"avg_score":   shop.AvgScore,
			"sold":        shop.Sold,
			"comments":    shop.Comments,
			"price_range": shop.PriceRange,
			"open_hours":  shop.OpenHours,
		}).Error

	if err != nil {
		return err
	}

	// Cache Aside 模式：删除缓存
	cacheKey := utils.ShopCachePrefix + string(rune(shop.ID))
	if err := utils.RedisClient.Del(ctx, cacheKey).Err(); err != nil {
		return err
	}

	return nil
}

// ListShopByType 按类型分页查询商户
func (s *ShopService) ListShopByType(ctx context.Context, typeId int64, page, size int) ([]*model.ShopDTO, int64, error) {
	// 1. 查数据库
	shops, total, err := s.shopDAO.ListByType(ctx, typeId, page, size)
	if err != nil {
		return nil, 0, err
	}

	// 2. 转换为DTO列表
	dtoList := make([]*model.ShopDTO, 0, len(shops))
	for _, shop := range shops {
		dto := &model.ShopDTO{
			ID:        shop.ID,
			Name:      shop.Name,
			TypeID:    shop.TypeID,
			Address:   shop.Address,
			AvgScore:  shop.AvgScore,
			OpenHours: shop.OpenHours,
		}
		dtoList = append(dtoList, dto)
	}

	return dtoList, total, nil
}
