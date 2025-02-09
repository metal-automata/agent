package servercontrol

import (
	"context"
	"strings"

	"github.com/metal-automata/agent/internal/ctrl"
	"github.com/metal-automata/agent/internal/device"
	"github.com/metal-automata/agent/internal/device/outofband"
	"github.com/metal-automata/agent/internal/model"
	"github.com/metal-automata/agent/internal/store"
	rctypes "github.com/metal-automata/rivets/condition"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	errUnsupportedAction = errors.New("unsupported action")
)

type OutofbandHandler struct {
	facilityCode,
	controllerID string
	bmc        device.OutofbandQueryor
	repository store.Repository
	publisher  ctrl.Publisher
	task       *model.ServerControlTask
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

func (o *OutofbandHandler) run(ctx context.Context, task *model.ServerControlTask) error {
	o.task = task
	queryor := outofband.NewDeviceQueryor(task.Server, o.logger)
	o.bmc = queryor

	defer func() {
		if err := queryor.Close(ctx); err != nil {
			o.logger.WithError(err).Warn("bmc connection close error")
		}
	}()

	switch task.Parameters.Action {
	case rctypes.GetPowerState:
		return o.powerState(ctx)
	case rctypes.PowerCycleBMC:
		return o.powerCycleBMC(ctx)
	case rctypes.SetPowerState:
		return o.setPowerState(ctx, task.Parameters.ActionParameter)
	case rctypes.SetNextBootDevice:
		return o.setNextBootDevice(
			ctx,
			task.Parameters.ActionParameter,
			task.Parameters.SetNextBootDevicePersistent,
			task.Parameters.SetNextBootDeviceEFI,
		)
	case rctypes.PxeBootPersistent:
		return o.pxeBootPersistent(ctx)
	default:
		return errors.Wrap(errUnsupportedAction, string(task.Parameters.Action))
	}
}

func (o *OutofbandHandler) publish(ctx context.Context, status string, state rctypes.State) error {
	o.task.State = state
	o.task.Status.Append(status)

	genTask, err := o.task.CopyAsGenericTask()
	if err != nil {
		o.logger.WithError(err).Error()
		return err
	}

	return o.publisher.Publish(ctx,
		genTask,
		false,
	)
}

// successful condition helper method
func (o *OutofbandHandler) successful(ctx context.Context, status string) error {
	if err := o.publish(ctx, status, rctypes.Succeeded); err != nil {
		o.logger.Warnf("failed to publish condition status: %s", status)
		return err
	}

	return nil
}

func (o *OutofbandHandler) powerState(ctx context.Context) error {
	state, err := o.bmc.PowerStatus(ctx)
	if err != nil {
		return errors.Wrap(err, "error identifying current power state")
	}

	return o.successful(ctx, state)
}

func (o *OutofbandHandler) powerCycleBMC(ctx context.Context) error {
	err := o.bmc.ResetBMC(ctx)
	if err != nil {
		return errors.Wrap(err, "error power cycling BMC")
	}

	return o.successful(ctx, "BMC power cycled successfully")
}

func (o *OutofbandHandler) setPowerState(ctx context.Context, newState string) error {
	// identify current power state
	state, err := o.bmc.PowerStatus(ctx)
	if err != nil {
		return errors.Wrap(err, "error identifying current power state")
	}

	err = o.publish(ctx, "identified current power state: "+state, rctypes.Active)
	if err != nil {
		return err
	}

	// for a power cycle - if a server is powered off, invoke power on instead of cycle
	if newState == "cycle" && strings.Contains(strings.ToLower(state), "off") {
		err = o.publish(ctx, "server was powered off, powering on", rctypes.Active)
		if err != nil {
			return err
		}

		err = o.bmc.SetPowerState(ctx, "on")
		if err != nil {
			return errors.Wrap(err, "server was powered off, failed to power on")
		}

		return o.successful(ctx, "server powered on successfully")
	}

	err = o.bmc.SetPowerState(ctx, newState)
	if err != nil {
		return errors.Wrap(err, "failed to set power state after power on")
	}

	return o.successful(ctx, "server power state set successful: "+o.task.Parameters.ActionParameter)
}

func (o *OutofbandHandler) setNextBootDevice(ctx context.Context, bootDevice string, persistent, efi bool) error {
	o.logger.WithFields(
		logrus.Fields{
			"persistent": persistent,
			"efi":        efi,
		}).Info("setting next boot device to: " + bootDevice)

	err := o.bmc.SetBootDevice(ctx, bootDevice, persistent, efi)
	if err != nil {
		return errors.Wrap(err, "error setting next boot device")
	}

	return o.successful(ctx, "next boot device set successfully: "+bootDevice)
}

// pxeBootPersistent sets up the server to pxe boot persistently
func (o *OutofbandHandler) pxeBootPersistent(ctx context.Context) error {
	if err := o.setNextBootDevice(ctx, "pxe", true, true); err != nil {
		return err
	}

	return o.bmc.SetPowerState(ctx, "on")
}
