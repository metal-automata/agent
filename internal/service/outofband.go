package service

import (
	"context"
	"sync"

	"github.com/metal-automata/agent/internal/firmware"
	"github.com/metal-automata/agent/internal/inventory"
	"github.com/metal-automata/agent/internal/model"
	"github.com/metal-automata/agent/internal/store"
	"github.com/metal-automata/agent/internal/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	"github.com/metal-automata/agent/internal/ctrl"
	rctypes "github.com/metal-automata/rivets/condition"
)

const (
	pkgName = "internal/worker"
)

type OobConditionTaskHandler struct {
	store          store.Repository
	syncWG         *sync.WaitGroup
	logger         *logrus.Logger
	facilityCode   string
	controllerID   string
	dryrun         bool
	faultInjection bool
}

// RunOutofband initializes the Out of band Condition handler and listens for events
func RunOutofband(
	ctx context.Context,
	dryrun,
	faultInjection bool,
	repository store.Repository,
	nc *ctrl.NatsController,
	logger *logrus.Logger,
) {
	ctx, span := otel.Tracer(pkgName).Start(
		ctx,
		"Run",
	)
	defer span.End()

	v := version.Current()
	logger.WithFields(
		logrus.Fields{
			"version":        v.AppVersion,
			"commit":         v.GitCommit,
			"branch":         v.GitBranch,
			"dry-run":        dryrun,
			"faultInjection": faultInjection,
		},
	).Info("OutOfBand agent running")

	handlerFactory := func() ctrl.TaskHandler {
		return &OobConditionTaskHandler{
			store:          repository,
			syncWG:         &sync.WaitGroup{},
			logger:         logger,
			dryrun:         dryrun,
			faultInjection: faultInjection,
			facilityCode:   nc.FacilityCode(),
			controllerID:   nc.ID(),
		}
	}

	if err := nc.ListenEvents(ctx, handlerFactory); err != nil {
		logger.Fatal(err)
	}
}

// HandleTask implements the ctrl.TaskHandler interface
func (h *OobConditionTaskHandler) HandleTask(
	ctx context.Context,
	genericTask *rctypes.Task[any, any],
	publisher ctrl.Publisher,
) error {
	if genericTask == nil {
		return errors.Wrap(model.ErrInitTask, "expected a generic Task object, got nil")
	}

	h.logger.WithFields(
		logrus.Fields{
			"conditionID": genericTask.ID.String(),
			"kind":        genericTask.Kind,
		},
	).Info("processing task..")

	switch genericTask.Kind {
	case rctypes.FirmwareInstall:
		fwHandler := firmware.NewHandler(
			h.facilityCode,
			h.controllerID,
			h.store,
			publisher,
		)

		if err := fwHandler.Run(ctx, genericTask, h.logger); err != nil {
			return err
		}

	case rctypes.Inventory:
		invHandler := inventory.NewHandler(
			h.facilityCode,
			h.controllerID,
			h.store,
			publisher,
		)

		if err := invHandler.Run(ctx, genericTask, h.logger); err != nil {
			return err
		}

	default:
		return errors.Wrap(model.ErrInitTask, "unsupport task kind: "+string(genericTask.Kind))
	}

	return nil
}
