package model

import (
	"encoding/json"
	"reflect"

	rctypes "github.com/metal-automata/rivets/condition"
	"github.com/mitchellh/copystructure"
	"github.com/pkg/errors"
)

// Inventory method is one of 'outofband' OR 'inband'
type InventoryMethod string

// Alias parameterized model.InventoryTask
type InventoryTask rctypes.Task[*rctypes.InventoryTaskParameters, any]

func (t *InventoryTask) SetState(s rctypes.State) {
	t.State = s
}

func (t *InventoryTask) MustMarshal() json.RawMessage {
	b, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}

	return b
}

func (t *InventoryTask) CopyAsGenericTask() (*rctypes.Task[any, any], error) {
	errTaskConv := errors.New("error in firmware install Task conversion")

	paramsJSON, err := t.Parameters.Marshal()
	if err != nil {
		return nil, errors.Wrap(errTaskConv, err.Error()+": Task.Parameters")
	}

	// deep copy fields referenced by pointer
	asset, err := copystructure.Copy(t.Server)
	if err != nil {
		return nil, errors.Wrap(errTaskConv, err.Error()+": Task.Server")
	}

	fault, err := copystructure.Copy(t.Fault)
	if err != nil {
		return nil, errors.Wrap(errTaskConv, err.Error()+": Task.Fault")
	}

	return &rctypes.Task[any, any]{
		StructVersion: t.StructVersion,
		ID:            t.ID,
		Kind:          t.Kind,
		State:         t.State,
		Status:        t.Status,
		Parameters:    paramsJSON,
		Fault:         fault.(*rctypes.Fault),
		FacilityCode:  t.FacilityCode,
		Server:        asset.(*rctypes.Server),
		WorkerID:      t.WorkerID,
		TraceID:       t.TraceID,
		SpanID:        t.SpanID,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
		CompletedAt:   t.CompletedAt,
	}, nil
}

func convInventoryTaskParams(params any) (*rctypes.InventoryTaskParameters, error) {
	errParamsConv := errors.New("error in Task.Parameters conversion")

	invParams := &rctypes.InventoryTaskParameters{}
	switch v := params.(type) {
	// When unpacked from a http request by the condition orc client,
	// Parameters are of this type.
	case map[string]interface{}:
		if err := invParams.MapStringInterfaceToStruct(v); err != nil {
			return nil, errors.Wrap(errParamsConv, err.Error())
		}
	// When received over NATS its of this type.
	case json.RawMessage:
		if err := invParams.Unmarshal(v); err != nil {
			return nil, errors.Wrap(errParamsConv, err.Error())
		}
	default:
		msg := "Task.Parameters expected to be one of map[string]interface{} or json.RawMessage, current type: " + reflect.TypeOf(params).String()
		return nil, errors.Wrap(errParamsConv, msg)
	}

	return invParams, nil
}

func CopyAsInventoryTask(task *rctypes.Task[any, any]) (*InventoryTask, error) {
	errTaskConv := errors.New("error in generic Task conversion")

	params, err := convInventoryTaskParams(task.Parameters)
	if err != nil {
		return nil, errors.Wrap(errTaskConv, err.Error())
	}

	// deep copy fields referenced by pointer
	asset, err := copystructure.Copy(task.Server)
	if err != nil {
		return nil, errors.Wrap(errTaskConv, err.Error()+": Task.Server")
	}

	fault, err := copystructure.Copy(task.Fault)
	if err != nil {
		return nil, errors.Wrap(errTaskConv, err.Error()+": Task.Fault")
	}

	return &InventoryTask{
		StructVersion: task.StructVersion,
		ID:            task.ID,
		Kind:          task.Kind,
		State:         task.State,
		Status:        task.Status,
		Parameters:    params,
		Fault:         fault.(*rctypes.Fault),
		FacilityCode:  task.FacilityCode,
		Server:        asset.(*rctypes.Server),
		WorkerID:      task.WorkerID,
		TraceID:       task.TraceID,
		SpanID:        task.SpanID,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
		CompletedAt:   task.CompletedAt,
	}, nil
}
