package inband

import (
	"context"

	"github.com/bmc-toolbox/common"
	"github.com/metal-automata/agent/internal/device"
	"github.com/metal-automata/ironlib"
	iactions "github.com/metal-automata/ironlib/actions"
	ironlibm "github.com/metal-automata/ironlib/model"
	iutils "github.com/metal-automata/ironlib/utils"
	"github.com/sirupsen/logrus"
)

type Client struct {
	logger *logrus.Logger
	dm     iactions.DeviceManager
}

// NewDeviceQueryor returns a server queryor that implements the DeviceQueryor interface
func NewDeviceQueryor(logger *logrus.Entry) device.InbandQueryor {
	return &Client{logger: logger.Logger}
}

func (s *Client) Inventory(ctx context.Context) (*common.Device, error) {
	dm, err := ironlib.New(s.logger)
	if err != nil {
		return nil, err
	}

	s.dm = dm

	disabledCollectors := []ironlibm.CollectorUtility{
		iutils.UefiFirmwareParserUtility,
		iutils.UefiVariableCollectorUtility,
		iutils.LsblkUtility,
	}

	return dm.GetInventory(ctx, iactions.WithDisabledCollectorUtilities(disabledCollectors))
}

// TODO: implement this method once the sandbox can pxe boot nodes
// Inventory implements the Queryor interface to collect inventory inband.
//
// The given asset object is updated with the collected information.
// func (i *Queryor) Inventory(ctx context.Context, asset *model.Asset) error {
// 	if !i.mock {
// 		var err error
//
// 		i.deviceManager, err = ironlib.New(i.logger.Logger)
// 		if err != nil {
// 			return err
// 		}
// 	}
//
// 	device, err := i.deviceManager.GetInventory(ctx)
// 	if err != nil {
// 		return err
// 	}
//
// 	device.Vendor = common.FormatVendorName(device.Vendor)
//
// 	// The "unknown" valued attributes here are to be filled in by the caller,
// 	// with the data from the inventory source when its available.
// 	asset.Inventory = device
// 	asset.Vendor = "unknown"
// 	asset.Model = "unknown"
// 	asset.Serial = "unknown"
//
// 	return nil
// }

func (s *Client) FirmwareInstall(ctx context.Context, component, vendor, model, _, updateFile string, force bool) error {
	params := &ironlibm.UpdateOptions{
		ForceInstall: force,
		Slug:         component,
		UpdateFile:   updateFile,
		Vendor:       vendor,
		Model:        model,
	}

	if s.dm == nil {
		dm, err := ironlib.New(s.logger)
		if err != nil {
			return err
		}

		s.dm = dm
	}

	return s.dm.InstallUpdates(ctx, params)
}

func (s *Client) FirmwareInstallRequirements(ctx context.Context, component, vendor, model string) (*ironlibm.UpdateRequirements, error) {
	if s.dm == nil {
		dm, err := ironlib.New(s.logger)
		if err != nil {
			return nil, err
		}

		s.dm = dm
	}

	return s.dm.UpdateRequirements(ctx, component, vendor, model)
}
