package firmware

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/bmc-toolbox/common"
	"github.com/metal-automata/agent/internal/device"
	"github.com/metal-automata/agent/internal/firmware/runner"
	"github.com/metal-automata/agent/internal/model"
	"github.com/metal-automata/agent/internal/store"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	rctypes "github.com/metal-automata/rivets/condition"
	// device inband
	devinb "github.com/metal-automata/agent/internal/device/inband"
	// device out-of-band
	devoob "github.com/metal-automata/agent/internal/device/outofband"
	// out-of-band action handler
	ahoob "github.com/metal-automata/agent/internal/firmware/outofband"
	// inband action handler
	ahinb "github.com/metal-automata/agent/internal/firmware/inband"
)

var (
	ErrSaveTask           = errors.New("error in saveTask transition taskHandler")
	ErrTaskTypeAssertion  = errors.New("error asserting Task type")
	errTaskQueryInventory = errors.New("error in task query inventory for installed firmware")
	errTaskPlanActions    = errors.New("error in task action planning")
)

// taskHandler implements the task.taskHandler interface to install firmware
//
// The taskHandler is instantiated to run a single task
type taskHandler struct {
	mode    model.RunMode
	resumed bool
	*runner.TaskHandlerContext
}

func newTaskHandler(
	mode model.RunMode,
	task *model.FirmwareTask,
	storage store.Repository,
	publisher runner.Publisher,
	logger *logrus.Entry,
) runner.TaskHandler {
	return &taskHandler{
		mode:    mode,
		resumed: task.State == model.StateActive,
		TaskHandlerContext: &runner.TaskHandlerContext{
			Task:      task,
			Publisher: publisher,
			Store:     storage,
			Logger:    logger,
		},
	}
}

func (t *taskHandler) Initialize(_ context.Context) error {
	if t.DeviceQueryor == nil {
		switch t.mode {
		case model.RunInband:
			t.DeviceQueryor = devinb.NewDeviceQueryor(t.Logger)
		case model.RunOutofband:
			t.DeviceQueryor = devoob.NewDeviceQueryor(t.Task.Server, t.Logger)
		}
	}

	return nil
}

func (t *taskHandler) Query(ctx context.Context) error {
	t.Logger.Debug("run query step")

	var err error
	var deviceCommon *common.Device
	switch t.mode {
	case model.RunInband:
		deviceCommon, err = t.inventoryInband(ctx)
		if err != nil {
			return err
		}
	case model.RunOutofband:
		deviceCommon, err = t.inventoryOutofband(ctx)
		if err != nil {
			return err
		}
	}

	if t.Task.Server.Vendor == "" {
		t.Task.Server.Vendor = deviceCommon.Vendor
	}

	if t.Task.Server.Model == "" {
		t.Task.Server.Model = common.FormatProductName(deviceCommon.Model)
	}

	server, err := t.Store.ConvertCommonDevice(t.Task.Parameters.AssetID, deviceCommon, model.InstallMethod(t.mode), true)
	if err != nil {
		return errors.Wrap(errTaskQueryInventory, err.Error())
	}

	// component inventory was identified
	if len(server.Components) > 0 {
		t.Task.Server.Components = server.Components

		return nil
	}

	return errors.Wrap(errTaskQueryInventory, "failed to query device component inventory")
}

func (t taskHandler) inventoryOutofband(ctx context.Context) (*common.Device, error) {
	if err := t.DeviceQueryor.(device.OutofbandQueryor).Open(ctx); err != nil {
		return nil, err
	}

	t.Task.Status.Append("connecting to device BMC")
	t.Publish(ctx)
	if err := t.DeviceQueryor.(device.OutofbandQueryor).Open(ctx); err != nil {
		return nil, err
	}

	t.Task.Status.Append("collecting inventory from device BMC")
	t.Publish(ctx)

	deviceCommon, err := t.DeviceQueryor.(device.OutofbandQueryor).Inventory(ctx)
	if err != nil {
		return nil, errors.Wrap(errTaskQueryInventory, err.Error())
	}

	return deviceCommon, nil
}

func (t taskHandler) inventoryInband(ctx context.Context) (*common.Device, error) {
	t.Task.Status.Append("collecting inventory from server")
	t.Publish(ctx)

	deviceCommon, err := t.DeviceQueryor.(device.InbandQueryor).Inventory(ctx)
	if err != nil {
		return nil, errors.Wrap(errTaskQueryInventory, err.Error())
	}

	return deviceCommon, nil
}

func (t *taskHandler) PlanActions(ctx context.Context) error {
	if t.resumed && len(t.Task.Data.ActionsPlanned) > 0 {
		return t.planResumedTask()
	}

	switch t.Task.Data.FirmwarePlanMethod {
	case model.FromFirmwareSet:
		return t.planFromFirmwareSet(ctx)
	case model.FromRequestedFirmware:
		return t.planFromFirmwareSlice(ctx)
	default:
		return errors.Wrap(errTaskPlanActions, "firmware plan method invalid: "+string(t.Task.Data.FirmwarePlanMethod))
	}
}

func (t *taskHandler) planFromFirmwareSlice(ctx context.Context) error {
	if len(t.Task.Parameters.Firmwares) == 0 {
		// we've been instructed to install firmwares listed in the parameters, but its empty
		return errors.Wrap(errTaskPlanActions, "planFromFirmwareSlice(): firmware set lacks any members")
	}

	// copy into slice of pointers
	applicable := []*rctypes.Firmware{}
	for idx := range t.Task.Parameters.Firmwares {
		applicable = append(applicable, &t.Task.Parameters.Firmwares[idx])
	}

	actions, err := t.planInstallActions(ctx, applicable)
	if err != nil {
		return err
	}

	t.Task.Data.ActionsPlanned = append(t.Task.Data.ActionsPlanned, actions...)

	return nil
}

// planFromFirmwareSet
func (t *taskHandler) planFromFirmwareSet(ctx context.Context) error {
	applicable, err := t.Store.FirmwareSetByID(ctx, t.Task.Parameters.FirmwareSetID)
	if err != nil {
		return errors.Wrap(errTaskPlanActions, err.Error())
	}

	if len(applicable) == 0 {
		// XXX: why not just short-circuit success here on the GIGO theory?
		return errors.Wrap(errTaskPlanActions, "planFromFirmwareSet(): firmware set lacks any members")
	}

	actions, err := t.planInstallActions(ctx, applicable)
	if err != nil {
		return err
	}

	t.Task.Data.ActionsPlanned = append(t.Task.Data.ActionsPlanned, actions...)

	return nil
}

func (t *taskHandler) planResumedTask() error {
	if t.mode == model.RunOutofband {
		return errors.Wrap(errTaskPlanActions, "resume task not (yet) supported on out-of-band firmware installs")
	}

	for _, action := range t.Task.Data.ActionsPlanned {
		if rctypes.StateIsComplete(action.State) {
			continue
		}

		actionCtx := &runner.ActionHandlerContext{
			TaskHandlerContext: t.TaskHandlerContext,
			Firmware:           &action.Firmware,
			First:              action.First,
			Last:               action.Last,
		}

		if err := ahinb.AssignStepHandlers(action, actionCtx); err != nil {
			return errors.Wrap(errTaskPlanActions, "failed to assign action step taskHandler: "+err.Error())
		}
	}

	return nil
}

// planInstall sets up the firmware install plan
//
// This returns a list of actions to added to the task and a list of action state machines for those actions.
func (t *taskHandler) planInstallActions(ctx context.Context, firmwares []*rctypes.Firmware) (model.Actions, error) {
	toInstall := []*rctypes.Firmware{}

	for _, fw := range firmwares {
		if t.mode == model.RunOutofband && !fw.InstallInband {
			toInstall = append(toInstall, fw)
		}

		if t.mode == model.RunInband && fw.InstallInband {
			toInstall = append(toInstall, fw)
		}
	}

	t.Logger.WithFields(logrus.Fields{
		"condition.id":             t.Task.ID,
		"requested.firmware.count": fmt.Sprintf("%d", len(toInstall)),
	}).Debug("checking against current inventory")

	// purge any firmware that are already installed
	if !t.Task.Parameters.ForceInstall {
		toInstall = t.removeFirmwareAlreadyAtDesiredVersion(toInstall)
	}

	if len(toInstall) == 0 {
		info := fmt.Sprintf("no %s firmware installs required", t.mode)
		t.Task.Status.Append(info)
		t.Publish(ctx)

		return nil, nil
	}

	// sort firmware in order of install
	t.sortFirmwareByInstallOrder(toInstall)

	actions := model.Actions{}
	// each firmware applicable results in an ActionPlan and an Action
	for idx, firmware := range toInstall {
		var actionHander runner.ActionHandler

		if t.mode == model.RunOutofband {
			if firmware.InstallInband {
				continue
			}

			actionHander = &ahoob.ActionHandler{}
		}

		if t.mode == model.RunInband {
			if !firmware.InstallInband {
				continue
			}

			actionHander = &ahinb.ActionHandler{}
		}

		actionCtx := &runner.ActionHandlerContext{
			TaskHandlerContext: t.TaskHandlerContext,
			Firmware:           firmware,
			First:              (idx == 0),
			Last:               (idx == len(toInstall)-1),
		}

		action, err := actionHander.ComposeAction(ctx, actionCtx)
		if err != nil {
			return nil, errors.Wrap(errTaskPlanActions, err.Error())
		}

		action.SetID(t.Task.ID.String(), firmware.Component, idx)
		action.SetState(model.StatePending)
		actions = append(actions, action)
	}

	var info string
	if len(actions) > 0 {
		info = fmt.Sprintf("planned firmware installs, method: %s, count: %d", t.mode, len(actions))
	} else {
		info = fmt.Sprintf("no %s firmware installs required", t.mode)
	}

	t.Task.Status.Append(info)
	t.Publish(ctx)
	t.Logger.Info(info)

	return actions, nil
}

func (t *taskHandler) sortFirmwareByInstallOrder(firmwares []*rctypes.Firmware) {
	sort.Slice(firmwares, func(i, j int) bool {
		slugi := strings.ToLower(firmwares[i].Component)
		slugj := strings.ToLower(firmwares[j].Component)
		return model.FirmwareInstallOrder[slugi] < model.FirmwareInstallOrder[slugj]
	})
}

// returns a list of firmware applicable and a list of causes for firmwares that were removed from the install list.
func (t *taskHandler) removeFirmwareAlreadyAtDesiredVersion(fws []*rctypes.Firmware) []*rctypes.Firmware {
	var toInstall []*rctypes.Firmware

	// TODO: The current invMap key is set to the component name,
	// This means if theres multiple Drives of different vendors only the last one in the
	// component list will be included. Consider a different approach where the key consists
	// of the name, model.
	//
	//	key := func(cmpName, cmpModel string) string {
	//		return fmt.Sprintf("%s.%s", strings.ToLower(cmpName), strings.ToLower(cmpModel))
	//	}

	invMap := make(map[string]string)
	for _, cmp := range t.Task.Server.Components {
		invMap[strings.ToLower(cmp.Name)] = cmp.InstalledFirmware.Version
	}

	fmtCause := func(component, cause, currentV, requestedV string) string {
		if currentV != "" && requestedV != "" {
			return fmt.Sprintf("[%s] %s, current=%s, requested=%s", component, cause, currentV, requestedV)
		}

		return fmt.Sprintf("[%s] %s", component, cause)
	}

	// XXX: this will drop firmware for components that are specified in
	// the firmware set but not in the inventory. This is consistent with the
	// desire of users to not require a force or a re-run to accomplish an
	// attainable goal.
	for _, fw := range fws {
		currentVersion, ok := invMap[strings.ToLower(fw.Component)]

		// skip install if current firmware version was not identified
		if currentVersion == "" && !t.Task.Parameters.ForceInstall {
			info := "Current firmware version returned empty, skipped install, use force to override"
			t.Task.Status.Append(
				fmtCause(
					fw.Component,
					info,
					currentVersion,
					fw.Version,
				),
			)

			t.Logger.WithFields(logrus.Fields{
				"component": fw.Component,
			}).Warn()

			continue
		}

		switch {
		case !ok:
			cause := "component not found in inventory"
			t.Logger.WithFields(logrus.Fields{
				"component": fw.Component,
			}).Warn(cause)

			t.Task.Status.Append(fmtCause(fw.Component, cause, "", ""))

		case strings.EqualFold(currentVersion, fw.Version):
			cause := "component firmware version equal"
			t.Logger.WithFields(logrus.Fields{
				"component": fw.Component,
				"version":   fw.Version,
			}).Debug(cause)

			t.Task.Status.Append(fmtCause(fw.Component, cause, currentVersion, fw.Version))

		default:
			t.Logger.WithFields(logrus.Fields{
				"component":         fw.Component,
				"installed.version": currentVersion,
				"mandated.version":  fw.Version,
			}).Debug("firmware queued for install")

			toInstall = append(toInstall, fw)

			t.Task.Status.Append(
				fmtCause(fw.Component, "firmware queued for install", currentVersion, fw.Version),
			)
		}
	}

	return toInstall
}

func (t *taskHandler) OnSuccess(ctx context.Context, _ *model.FirmwareTask) {
	if t.mode == model.RunInband || t.DeviceQueryor == nil {
		return
	}

	if err := t.DeviceQueryor.(device.OutofbandQueryor).Close(ctx); err != nil {
		t.Logger.WithFields(logrus.Fields{"err": err.Error()}).Warn("device logout error")
	}
}

func (t *taskHandler) OnFailure(ctx context.Context, _ *model.FirmwareTask) {
	if t.mode == model.RunInband || t.DeviceQueryor == nil {
		return
	}

	if err := t.DeviceQueryor.(device.OutofbandQueryor).Close(ctx); err != nil {
		t.Logger.WithFields(logrus.Fields{"err": err.Error()}).Warn("device logout error")
	}
}

func (t *taskHandler) Publish(ctx context.Context) {
	//nolint:errcheck // method called logs errors if any
	_ = t.Publisher.Publish(ctx, t.Task)
}
