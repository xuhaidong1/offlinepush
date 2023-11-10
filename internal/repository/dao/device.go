package dao

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var ErrDuplicateRecord = errors.New("ErrDuplicateRecord")

type Device struct {
	ID int64 `gorm:"primaryKey,autoIncrement"`
	// Type  string `gorm:"not null;type:varchar(16);index;uniqueIndex:type_cid"`
	// CID   int64  `gorm:"not null;uniqueIndex:type_cid"`
	Type  string `gorm:"not null;type:varchar(16);"`
	CID   int64  `gorm:"not null"`
	Ctime int64  `gorm:"not null;default:0;comment:创建时间"`
	Utime int64  `gorm:"not null;default:0;comment:更新时间UTC毫秒数"`
}

type DeviceDAO interface {
	Insert(ctx context.Context, d Device) error
	BatchInsert(ds []Device) error
	FindDevices(ctx context.Context, typ string) ([]Device, error)
	FindDevicesLimit(ctx context.Context, typ string, limit int) ([]Device, error)
}

type gormDeviceDAO struct {
	db *gorm.DB
}

func NewGormDeviceDAO(db *gorm.DB) DeviceDAO {
	return &gormDeviceDAO{db: db}
}

func (dao *gormDeviceDAO) Insert(ctx context.Context, d Device) error {
	now := time.Now().UnixMilli()
	d.Utime = now
	d.Ctime = now
	err := dao.db.WithContext(ctx).Create(&d).Error
	me := new(mysql.MySQLError)
	if ok := errors.As(err, &me); ok {
		const uniqueConflicts uint16 = 1062
		if me.Number == uniqueConflicts {
			return ErrDuplicateRecord
		}
	}
	return err
}

func (dao *gormDeviceDAO) FindDevices(ctx context.Context, typ string) ([]Device, error) {
	var res []Device
	err := dao.db.WithContext(ctx).Where("type = ?", typ).
		Find(&res).Error
	return res, err
}

func (dao *gormDeviceDAO) FindDevicesLimit(ctx context.Context, typ string, limit int) ([]Device, error) {
	var res []Device
	err := dao.db.WithContext(ctx).Where("type = ?", typ).Limit(limit).
		Find(&res).Error
	return res, err
}

func (dao *gormDeviceDAO) BatchInsert(ds []Device) error {
	return dao.db.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < len(ds); i++ {
			err := dao.insert(tx, ds[i])
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (dao *gormDeviceDAO) insert(tx *gorm.DB, d Device) error {
	return tx.Create(&d).Error
}
