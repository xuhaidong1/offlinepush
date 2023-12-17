package repository

import (
	"context"

	"github.com/xuhaidong1/offlinepush/internal/domain"
	"github.com/xuhaidong1/offlinepush/internal/repository/dao"
)

type Repository interface {
	Create(ctx context.Context, d domain.Device) error
	BatchInsert(ds []domain.Device) error
	FindDevices(ctx context.Context, typ string) ([]domain.Device, error)
	FindDevicesLimit(ctx context.Context, typ string, limit int) ([]domain.Device, error)
}

type repository struct {
	dao dao.DeviceDAO
}

func NewRepository(dao dao.DeviceDAO) Repository {
	return &repository{dao: dao}
}

func (r *repository) Create(ctx context.Context, d domain.Device) error {
	return r.dao.Insert(ctx, r.domainToEntity(d))
}

func (r *repository) BatchInsert(ds []domain.Device) error {
	dvs := make([]dao.Device, 0, len(ds))
	for _, d := range ds {
		dvs = append(dvs, r.domainToEntity(d))
	}
	return r.dao.BatchInsert(dvs)
}

func (r *repository) FindDevices(ctx context.Context, typ string) ([]domain.Device, error) {
	devices, err := r.dao.FindDevices(ctx, typ)
	if err != nil {
		return nil, err
	}
	dvs := make([]domain.Device, 0, len(devices))
	for _, d := range devices {
		dvs = append(dvs, r.entityToDomain(d))
	}
	return dvs, nil
}

func (r *repository) FindDevicesLimit(ctx context.Context, typ string, limit int) ([]domain.Device, error) {
	devices, err := r.dao.FindDevicesLimit(ctx, typ, limit)
	if err != nil {
		return nil, err
	}
	dvs := make([]domain.Device, 0, len(devices))
	for _, d := range devices {
		dvs = append(dvs, r.entityToDomain(d))
	}
	return dvs, nil
}

func (r *repository) domainToEntity(d domain.Device) dao.Device {
	return dao.Device{
		Type: d.Type,
		CID:  d.ID,
	}
}

func (r *repository) entityToDomain(d dao.Device) domain.Device {
	return domain.Device{
		Type: d.Type,
		ID:   d.CID,
	}
}
