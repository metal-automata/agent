package inventory

import (
	"context"

	"github.com/bmc-toolbox/common"
	"github.com/metal-automata/agent/internal/ctrl"
	"github.com/metal-automata/agent/internal/model"
	"github.com/metal-automata/agent/internal/store"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	rctypes "github.com/metal-automata/rivets/condition"
)

var (
	ErrBiosCfgCollect   = errors.New("error collecting BIOS configuration")
	ErrInventoryCollect = errors.New("error collecting inventory data")
)

type collection struct {
	inventory *common.Device
	biosCfg   map[string]string
}

type Handler struct {
	facilityCode,
	controllerID string
	repository store.Repository
	publisher  ctrl.Publisher
}

func NewHandler(facilityCode, controllerID string, repository store.Repository, publisher ctrl.Publisher) *Handler {
	return &Handler{
		facilityCode: facilityCode,
		controllerID: controllerID,
		repository:   repository,
		publisher:    publisher,
	}
}

func (h *Handler) Run(ctx context.Context, genericTask *rctypes.Task[any, any], l *logrus.Logger) error {
	task, runMode, ctxLogger, err := h.initTask(ctx, genericTask, l)
	if err != nil {
		l.WithFields(logrus.Fields{
			"assetID":      genericTask.Server.UUID.String(),
			"conditionID":  genericTask.ID,
			"controllerID": h.controllerID,
			"err":          err.Error(),
			"mode":         runMode,
		}).Error("task init error")

		return err
	}

	ctxLogger.WithField("mode", runMode).Info("running task for device")

	switch runMode {
	case model.RunInband:
	case model.RunOutofband:
		h := NewOutofbandHandler(
			h.facilityCode,
			h.controllerID,
			h.repository,
			h.publisher,
			ctxLogger,
		)

		collection, err := h.Collect(ctx, task)
		if err != nil {
			ctxLogger.WithError(err).Error("Collect() returned error")
			return err
		}

		if errInv := h.repository.SetComponentInventory(
			ctx,
			task.Server.UUID,
			collection.inventory,
			model.InstallMethod(model.RunOutofband),
		); errInv != nil {
			ctxLogger.WithError(errInv).Error("SetComponentInventory() returned error")
			return errInv
		}
	}

	ctxLogger.Info("task for device completed")
	return nil
}

func (h *Handler) initTask(ctx context.Context, genericTask *rctypes.Task[any, any], l *logrus.Logger) (*model.InventoryTask, model.RunMode, *logrus.Entry, error) {
	// prepare new logger for handler
	logger := logrus.New()
	logger.Formatter = l.Formatter
	logger.Level = l.Level

	task, err := model.CopyAsInventoryTask(genericTask)
	if err != nil {
		return nil, "", nil, errors.Wrap(model.ErrInitTask, err.Error())
	}

	// fetch server inventory from inventory store
	//
	// TODO: remove this lookup
	server, err := h.repository.AssetByID(ctx, task.Parameters.AssetID.String())
	if err != nil {
		return nil, "", nil, errors.Wrap(model.ErrInitTask, err.Error())
	}

	task.Server = server
	task.FacilityCode = h.facilityCode
	task.WorkerID = h.controllerID

	ctxLogger := logger.WithFields(
		logrus.Fields{
			"conditionID":  task.ID.String(),
			"controllerID": h.controllerID,
			"assetID":      server.UUID.String(),
			"bmc":          server.BMC.IPAddress,
		},
	)

	switch task.Parameters.Method {
	case rctypes.OutofbandInventory:
		return task, model.RunOutofband, ctxLogger, nil
	}

	return nil, "", nil, errors.Wrap(model.ErrInitTask, "unsupported task run mode: "+string(task.Parameters.Method))
}
