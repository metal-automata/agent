package inventory

import (
	"context"
	"fmt"
	"strings"

	"github.com/metal-automata/agent/internal/ctrl"
	"github.com/metal-automata/agent/internal/device/outofband"
	"github.com/metal-automata/agent/internal/model"
	"github.com/metal-automata/agent/internal/store"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type OutofbandHandler struct {
	facilityCode,
	controllerID string
	repository store.Repository
	publisher  ctrl.Publisher
	logger     *logrus.Entry
}

func NewOutofbandHandler(facilityCode, controllerID string, repository store.Repository, publisher ctrl.Publisher, l *logrus.Entry) *OutofbandHandler {
	return &OutofbandHandler{
		facilityCode: facilityCode,
		controllerID: controllerID,
		repository:   repository,
		publisher:    publisher,
		logger:       l,
	}
}

// nolint:revive // TODO: fix up method and then purge nolint
func (c *OutofbandHandler) Collect(ctx context.Context, task *model.InventoryTask) (*collection, error) {
	queryor := outofband.NewDeviceQueryor(task.Server, c.logger)

	defer func() {
		if err := queryor.Close(ctx); err != nil {
			c.logger.WithError(err).Warn("bmc connection close error")
		}
	}()

	collected := &collection{}

	// collect inventory
	commonDevice, err := queryor.Inventory(ctx)
	if err != nil {
		return nil, collectionError("inventory", err)
	}

	collected.inventory = commonDevice

	// TODO: provide BIOS configuration storage in fleetdb
	// collect BIOS configurations
	// biosCfg, err := queryor.BiosConfiguration(ctx)
	// if err != nil {
	// 	errB := collectionError("bioscfg", err)
	// 	c.logger.WithError(errB).Warn("bios configuration collection error")
	// }
	// collected.biosCfg = biosCfg

	return collected, nil
}

func collectionError(kind string, err error) error {
	// nolint:err113 // dynamic error here defined on purpose
	collectionErr := fmt.Errorf("error in %s collection", kind)

	switch {
	case strings.Contains(err.Error(), "no compatible System Odata IDs identified"):
		// device provides a redfish API, but BIOS configuration export isn't supported in the current redfish library
		return errors.Wrap(collectionErr, "redfish_incompatible: no compatible System Odata IDs identified")
	case strings.Contains(err.Error(), "no BiosConfigurationGetter implementations found"):
		// no means to export BIOS configuration were found
		return errors.Wrap(collectionErr, "device not supported")
	default:
		return errors.Wrap(collectionErr, err.Error())
	}
}
