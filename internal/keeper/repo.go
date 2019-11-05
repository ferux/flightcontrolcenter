package keeper

import (
	"context"

	"github.com/ferux/flightcontrolcenter/internal/model"
)

type Repo interface {
	Get(ctx context.Context, deviceID model.DeviceID) (model.Device, error)
	GetByUUID(ctx context.Context, deviceUUID string) (model.Device, error)
	Insert(ctx context.Context, device model.Device) (model.DeviceID, error)
	Update(ctx context.Context, device model.Device) error
	Mark(ctx context.Context, deviceID model.DeviceID, state model.DeviceState) error
	MarkAll(ctx context.Context, state model.DeviceState) error
}
