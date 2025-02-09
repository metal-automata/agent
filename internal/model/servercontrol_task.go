package model

import (
	"encoding/json"
	"reflect"

	rctypes "github.com/metal-automata/rivets/condition"
	"github.com/mitchellh/copystructure"
	"github.com/pkg/errors"
)

// Alias parameterized model.ServerControlTask
type ServerControlTask rctypes.Task[*rctypes.ServerControlTaskParameters, json.RawMessage]

func (t *ServerControlTask) SetState(s rctypes.State) {
	t.State = s
}

func (t *ServerControlTask) MustMarshal() json.RawMessage {
	b, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}

	return b
}

func (t *ServerControlTask) CopyAsGenericTask() (*rctypes.Task[any, any], error) {
	errTaskConv := errors.New("error in inventory Task conversion")

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

func convServerControlTaskParams(params any) (*rctypes.ServerControlTaskParameters, error) {
	errParamsConv := errors.New("error in Task.Parameters conversion")

	scParams := &rctypes.ServerControlTaskParameters{}
	switch v := params.(type) {
	// When unpacked from a http request by the condition orc client,
	// Parameters are of this type.
	case map[string]interface{}:
		if err := scParams.MapStringInterfaceToStruct(v); err != nil {
			return nil, errors.Wrap(errParamsConv, err.Error())
		}
	// When received over NATS its of this type.
	case json.RawMessage:
		if err := scParams.Unmarshal(v); err != nil {
			return nil, errors.Wrap(errParamsConv, err.Error())
		}
	default:
		msg := "Task.Parameters expected to be one of map[string]interface{} or json.RawMessage, current type: " + reflect.TypeOf(params).String()
		return nil, errors.Wrap(errParamsConv, msg)
	}

	return scParams, nil
}

func CopyAsServerControlTask(task *rctypes.Task[any, any]) (*ServerControlTask, error) {
	errTaskConv := errors.New("error in generic Task conversion")

	params, err := convServerControlTaskParams(task.Parameters)
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

	return &ServerControlTask{
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
