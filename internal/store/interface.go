package store

import (
	"context"

	"github.com/bmc-toolbox/common"
	"github.com/google/uuid"
	"github.com/metal-automata/agent/internal/model"
	fleetdbapi "github.com/metal-automata/fleetdb/pkg/api/v1"
	rctypes "github.com/metal-automata/rivets/condition"
)

type Repository interface {
	// AssetByID returns asset.
	AssetByID(ctx context.Context, id string) (*rctypes.Server, error)

	FirmwareSetByID(ctx context.Context, id uuid.UUID) ([]*rctypes.Firmware, error)

	// FirmwareByDeviceVendorModel returns the firmware for the device vendor, model.
	FirmwareByDeviceVendorModel(ctx context.Context, deviceVendor, deviceModel string) ([]*rctypes.Firmware, error)

	// Converts from the common.Device to the fleetdbapi.Server type
	//
	// checkComponentSlug when set will cause the convertor to verify the components are of a valid ComponentSlugType in fleetdbapi
	// this check should be *enabled* for when the converted inventory is to be stored in fleetdb.
	ConvertCommonDevice(serverID uuid.UUID, hw *common.Device, collectionMethod model.CollectionMethod, checkComponentSlug bool) (*rctypes.Server, error)

	// Initialize or update component inventory
	SetComponentInventory(ctx context.Context, serverID uuid.UUID, components fleetdbapi.ServerComponentSlice, initialized bool, method model.CollectionMethod) error
}
