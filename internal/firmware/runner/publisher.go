package runner

import (
	"context"

	"github.com/metal-automata/agent/internal/ctrl"
	"github.com/metal-automata/agent/internal/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	ErrPublishStatus = errors.New("error in publish Condition status")
	ErrPublishTask   = errors.New("error in publish Condition Task")
)

// Publisher defines methods to publish task information.
type Publisher interface {
	Publish(ctx context.Context, task *model.FirmwareTask) error
}

// StatusPublisher implements the Publisher interface
// to wrap the condition controller publish method
type StatusPublisher struct {
	logger *logrus.Entry
	cp     ctrl.Publisher
}

func NewTaskStatusPublisher(logger *logrus.Entry, cp ctrl.Publisher) Publisher {
	return &StatusPublisher{
		logger,
		cp,
	}
}

func (s *StatusPublisher) Publish(ctx context.Context, task *model.FirmwareTask) error {
	genericTask, err := task.CopyAsGenericTask()
	if err != nil {
		err = errors.Wrap(ErrPublishTask, err.Error())
		s.logger.WithError(err).Warn("Task publish error")

		return err
	}

	if genericTask.Server.BMC != nil {
		// overwrite credentials before this gets written back to the repository
		genericTask.Server.BMC.IPAddress = ""
		genericTask.Server.BMC.Password = ""
		genericTask.Server.BMC.Username = ""
	}

	if err := s.cp.Publish(ctx, genericTask, false); err != nil {
		err = errors.Wrap(ErrPublishStatus, err.Error())
		s.logger.WithError(err).Error("Condition status publish error")

		return err
	}

	s.logger.Trace("Condition Status publish successful")
	return nil
}
