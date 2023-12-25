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
	ID   int64  `gorm:"primaryKey,autoIncrement"`
	Type string `gorm:"not null;type:varchar(16);index;unique_Index:type_cid"`
	// Type  string `gorm:"not null;type:varchar(16);"`
	CID   int64 `gorm:"not null;unique_Index:type_cid"`
	Ctime int64 `gorm:"not null;default:0;comment:创建时间"`
	Utime int64 `gorm:"not null;default:0;comment:更新时间UTC毫秒数"`
}

type DeviceDAO interface {
	Insert(ctx context.Context, d Device) error
	BatchInsert(ds []Device) error
	FindDevices(ctx context.Context, typ string) ([]Device, error)
	FindDevicesPage(ctx context.Context, typ string, limit int, cursor *Cursor) ([]Device, error)
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

//func (dao *gormDeviceDAO) FindDevicesPage(ctx context.Context, typ string, limit, offset int) ([]Device, error) {
//	var res []Device
//	err := dao.db.WithContext(ctx).Where("type = ?", typ).Limit(limit).Offset(offset).
//		Find(&res).Error
//	return res, err
//}

func (dao *gormDeviceDAO) FindDevicesPage(ctx context.Context, typ string, limit int, cursor *Cursor) ([]Device, error) {
	var res []Device
	err := dao.db.WithContext(ctx).Select("c_id").
		Where("type = ? AND c_id > ?", typ, cursor.Get()).Limit(limit).
		Find(&res).Error
	if len(res) > 0 {
		cursor.Set(res[len(res)-1].CID)
	}
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

type Cursor struct {
	data int64
}

func (c *Cursor) Get() int64 {
	return c.data
}

func (c *Cursor) Set(d int64) {
	c.data = d
}
