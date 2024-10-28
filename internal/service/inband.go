package service

import (
	"context"

	"github.com/metal-automata/agent/internal/ctrl"
	"github.com/metal-automata/agent/internal/firmware"
	"github.com/metal-automata/agent/internal/model"
	"github.com/metal-automata/agent/internal/store"
	"github.com/metal-automata/agent/internal/version"
	rctypes "github.com/metal-automata/rivets/condition"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

// implements the controller.TaskHandler interface
type InbandConditionTaskHandler struct {
	store          store.Repository
	logger         *logrus.Logger
	facilityCode   string
	dryrun         bool
	faultInjection bool
}

// RunInband initializes the inband agent
func RunInband(
	ctx context.Context,
	dryrun,
	faultInjection bool,
	facilityCode string,
	repository store.Repository,
	nc *ctrl.HTTPController,
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
	).Info("Inband agent running")

	inbHandler := InbandConditionTaskHandler{
		store:          repository,
		logger:         logger,
		dryrun:         dryrun,
		faultInjection: faultInjection,
		facilityCode:   facilityCode,
	}

	if err := nc.Run(ctx, &inbHandler); err != nil {
		logger.Fatal(err)
	}
}

// Handle implements the ctrl.ConditionHandler interface
func (h *InbandConditionTaskHandler) HandleTask(
	ctx context.Context,
	genericTask *rctypes.Task[any, any],
	publisher ctrl.Publisher,
) error {
	if genericTask == nil {
		return errors.Wrap(model.ErrInitTask, "expected a generic Task object, got nil")
	}

	switch genericTask.Kind {
	case rctypes.FirmwareInstall:
		fwHandler := firmware.NewHandler(
			h.facilityCode,
			"",
			h.store,
			publisher,
		)

		return fwHandler.Run(ctx, genericTask, h.logger)

	case rctypes.Inventory:
	}

	return nil
}
