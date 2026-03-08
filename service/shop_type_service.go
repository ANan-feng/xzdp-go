package service

import (
	"context"
	"xzdp-go/dao"
	"xzdp-go/model"
)

// ShopTypeService 商户类型业务逻辑层
type ShopTypeService struct {
	shopTypeDAO *dao.ShopTypeDAO
}

// NewShopTypeService 创建商户类型服务实例
func NewShopTypeService() *ShopTypeService {
	return &ShopTypeService{
		shopTypeDAO: dao.NewShopTypeDAO(),
	}
}

// GetShopTypeById 根据ID查询商户类型
func (s *ShopTypeService) GetShopTypeById(ctx context.Context, typeId int64) (*model.ShopTypeDTO, error) {
	shopType, err := s.shopTypeDAO.GetById(ctx, typeId)
	if err != nil {
		return nil, err
	}
	if shopType == nil {
		return nil, nil
	}
	// 转换为DTO
	dto := &model.ShopTypeDTO{
		ID:   shopType.ID,
		Name: shopType.Name,
		Icon: shopType.Icon,
		Sort: shopType.Sort,
	}
	return dto, nil
}

// ListAllShopTypes 查询所有商户类型（按sort降序）
func (s *ShopTypeService) ListAllShopTypes(ctx context.Context) ([]*model.ShopTypeDTO, error) {
	types, err := s.shopTypeDAO.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	// 转换为DTO列表
	dtoList := make([]*model.ShopTypeDTO, 0, len(types))
	for _, t := range types {
		dto := &model.ShopTypeDTO{
			ID:   t.ID,
			Name: t.Name,
			Icon: t.Icon,
			Sort: t.Sort,
		}
		dtoList = append(dtoList, dto)
	}
	return dtoList, nil
}

// CreateShopType 新增商户类型
func (s *ShopTypeService) CreateShopType(ctx context.Context, shopType *model.ShopType) error {
	return s.shopTypeDAO.Create(ctx, shopType)
}

// UpdateShopType 更新商户类型
func (s *ShopTypeService) UpdateShopType(ctx context.Context, shopType *model.ShopType) error {
	return s.shopTypeDAO.Update(ctx, shopType)
}

// DeleteShopType 根据ID删除商户类型
func (s *ShopTypeService) DeleteShopType(ctx context.Context, typeId int64) error {
	return s.shopTypeDAO.Delete(ctx, typeId)
}
