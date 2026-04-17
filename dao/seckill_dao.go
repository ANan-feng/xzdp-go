package dao

import (
	"context"
	"xzdp-go/model"

	"gorm.io/gorm"
)

type SeckillDAO struct {
	db *gorm.DB
}

func NewSeckillDAO(db *gorm.DB) *SeckillDAO {
	return &SeckillDAO{db: db}
}

// GetSeckillVoucherByID 查询秒杀优惠券（关联主表）
// ✅ 修复：表名改为 seckill_vouchers
func (dao *SeckillDAO) GetSeckillVoucherByID(ctx context.Context, voucherID int64) (*model.SeckillVouchers, *model.Voucher, error) {
	var skVoucher model.SeckillVouchers
	var voucher model.Voucher

	err := dao.db.WithContext(ctx).
		Table("seckill_vouchers").                                                   // ✅ 这里修复复数
		Joins("LEFT JOIN voucher ON seckill_vouchers.voucher_id = voucher.id").      // ✅ 这里修复复数
		Where("seckill_vouchers.voucher_id = ?", voucherID).                         // ✅ 这里修复复数
		Select("seckill_vouchers.*, voucher.shop_id, voucher.type, voucher.status"). // ✅ 这里修复复数
		First(&skVoucher).Error
	if err != nil {
		return nil, nil, err
	}

	// 单独查询voucher
	err = dao.db.WithContext(ctx).
		Table("voucher").
		Where("id = ?", voucherID).
		First(&voucher).Error

	return &skVoucher, &voucher, nil
}

// CreateSeckillOrder 创建秒杀订单（乐观锁防超卖）
// ✅ 修复：表名改为 seckill_vouchers
func (dao *SeckillDAO) CreateSeckillOrder(ctx context.Context, order *model.SeckillOrders, voucherID int64) error {
	return dao.db.Transaction(func(tx *gorm.DB) error {
		// 乐观锁扣库存
		res := tx.Model(&model.SeckillVouchers{}).
			Where("voucher_id = ? AND stock > 0", voucherID).
			Update("stock", gorm.Expr("stock - 1"))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		// 创建订单
		return tx.Create(order).Error
	})
}

// CheckUserOrderExist 检查用户是否已下单（一人一单）
func (dao *SeckillDAO) CheckUserOrderExist(ctx context.Context, userID, voucherID int64) (bool, error) {
	var count int64
	err := dao.db.WithContext(ctx).
		Model(&model.SeckillOrders{}).
		Where("user_id = ? AND voucher_id = ?", userID, voucherID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
