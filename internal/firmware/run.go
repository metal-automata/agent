package firmware

import (
	"context"

	"github.com/metal-automata/agent/internal/ctrl"
	"github.com/metal-automata/agent/internal/firmware/runner"
	"github.com/metal-automata/agent/internal/model"
	"github.com/metal-automata/agent/internal/store"
	rctypes "github.com/metal-automata/rivets/condition"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

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
			"assetID":      task.Parameters.AssetID.String(),
			"conditionID":  task.ID,
			"controllerID": h.controllerID,
			"err":          err.Error(),
			"mode":         runMode,
		}).Error("task init error")

		return err
	}

	handler := newTaskHandler(
		runMode,
		task,
		h.repository,
		runner.NewTaskStatusPublisher(ctxLogger, h.publisher),
		ctxLogger,
	)

	// init runner
	r := runner.New(ctxLogger)

	ctxLogger.WithField("mode", runMode).Info("running task for device")
	if err := r.RunTask(ctx, task, handler); err != nil {
		ctxLogger.WithError(err).Error("task for device failed")
		return err
	}

	ctxLogger.Info("task for device completed")

	return nil
}

func (h *Handler) initTask(ctx context.Context, genericTask *rctypes.Task[any, any], l *logrus.Logger) (*model.FirmwareTask, model.RunMode, *logrus.Entry, error) {
	// prepare new logger for handler
	logger := logrus.New()
	logger.Formatter = l.Formatter
	logger.Level = l.Level

	task, err := model.CopyAsFirmwareTask(genericTask)
	if err != nil {
		return nil, "", nil, errors.Wrap(model.ErrInitTask, err.Error())
	}

	switch task.Kind {
	case rctypes.FirmwareInstall:
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
				"serverID":     server.UUID.String(),
				"bmc":          server.BMC.IPAddress,
			},
		)

		return task, model.RunOutofband, ctxLogger, nil

	case rctypes.FirmwareInstallInband:
		ctxLogger := l.WithFields(
			logrus.Fields{
				"conditionID": task.ID.String(),
				"serverID":    task.Server.UUID.String(),
			},
		)

		return task, model.RunInband, ctxLogger, nil

	default:
		return nil, "", nil, errors.Wrap(model.ErrInitTask, "unsupported task kind: "+string(task.Kind))
	}
}
