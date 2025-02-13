// Code generated by mockery v2.42.1. DO NOT EDIT.

package runner

import (
	context "context"

	model "github.com/metal-automata/agent/internal/model"
	mock "github.com/stretchr/testify/mock"
)

// MockTaskHandler is an autogenerated mock type for the TaskHandler type
type MockTaskHandler struct {
	mock.Mock
}

type MockTaskHandler_Expecter struct {
	mock *mock.Mock
}

func (_m *MockTaskHandler) EXPECT() *MockTaskHandler_Expecter {
	return &MockTaskHandler_Expecter{mock: &_m.Mock}
}

// Initialize provides a mock function with given fields: ctx
func (_m *MockTaskHandler) Initialize(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Initialize")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockTaskHandler_Initialize_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Initialize'
type MockTaskHandler_Initialize_Call struct {
	*mock.Call
}

// Initialize is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockTaskHandler_Expecter) Initialize(ctx interface{}) *MockTaskHandler_Initialize_Call {
	return &MockTaskHandler_Initialize_Call{Call: _e.mock.On("Initialize", ctx)}
}

func (_c *MockTaskHandler_Initialize_Call) Run(run func(ctx context.Context)) *MockTaskHandler_Initialize_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockTaskHandler_Initialize_Call) Return(_a0 error) *MockTaskHandler_Initialize_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockTaskHandler_Initialize_Call) RunAndReturn(run func(context.Context) error) *MockTaskHandler_Initialize_Call {
	_c.Call.Return(run)
	return _c
}

// OnFailure provides a mock function with given fields: ctx, task
func (_m *MockTaskHandler) OnFailure(ctx context.Context, task *model.FirmwareTask) {
	_m.Called(ctx, task)
}

// MockTaskHandler_OnFailure_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'OnFailure'
type MockTaskHandler_OnFailure_Call struct {
	*mock.Call
}

// OnFailure is a helper method to define mock.On call
//   - ctx context.Context
//   - task *model.FirmwareTask
func (_e *MockTaskHandler_Expecter) OnFailure(ctx interface{}, task interface{}) *MockTaskHandler_OnFailure_Call {
	return &MockTaskHandler_OnFailure_Call{Call: _e.mock.On("OnFailure", ctx, task)}
}

func (_c *MockTaskHandler_OnFailure_Call) Run(run func(ctx context.Context, task *model.FirmwareTask)) *MockTaskHandler_OnFailure_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*model.FirmwareTask))
	})
	return _c
}

func (_c *MockTaskHandler_OnFailure_Call) Return() *MockTaskHandler_OnFailure_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockTaskHandler_OnFailure_Call) RunAndReturn(run func(context.Context, *model.FirmwareTask)) *MockTaskHandler_OnFailure_Call {
	_c.Call.Return(run)
	return _c
}

// OnSuccess provides a mock function with given fields: ctx, task
func (_m *MockTaskHandler) OnSuccess(ctx context.Context, task *model.FirmwareTask) {
	_m.Called(ctx, task)
}

// MockTaskHandler_OnSuccess_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'OnSuccess'
type MockTaskHandler_OnSuccess_Call struct {
	*mock.Call
}

// OnSuccess is a helper method to define mock.On call
//   - ctx context.Context
//   - task *model.FirmwareTask
func (_e *MockTaskHandler_Expecter) OnSuccess(ctx interface{}, task interface{}) *MockTaskHandler_OnSuccess_Call {
	return &MockTaskHandler_OnSuccess_Call{Call: _e.mock.On("OnSuccess", ctx, task)}
}

func (_c *MockTaskHandler_OnSuccess_Call) Run(run func(ctx context.Context, task *model.FirmwareTask)) *MockTaskHandler_OnSuccess_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*model.FirmwareTask))
	})
	return _c
}

func (_c *MockTaskHandler_OnSuccess_Call) Return() *MockTaskHandler_OnSuccess_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockTaskHandler_OnSuccess_Call) RunAndReturn(run func(context.Context, *model.FirmwareTask)) *MockTaskHandler_OnSuccess_Call {
	_c.Call.Return(run)
	return _c
}

// PlanActions provides a mock function with given fields: ctx
func (_m *MockTaskHandler) PlanActions(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for PlanActions")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockTaskHandler_PlanActions_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PlanActions'
type MockTaskHandler_PlanActions_Call struct {
	*mock.Call
}

// PlanActions is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockTaskHandler_Expecter) PlanActions(ctx interface{}) *MockTaskHandler_PlanActions_Call {
	return &MockTaskHandler_PlanActions_Call{Call: _e.mock.On("PlanActions", ctx)}
}

func (_c *MockTaskHandler_PlanActions_Call) Run(run func(ctx context.Context)) *MockTaskHandler_PlanActions_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockTaskHandler_PlanActions_Call) Return(_a0 error) *MockTaskHandler_PlanActions_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockTaskHandler_PlanActions_Call) RunAndReturn(run func(context.Context) error) *MockTaskHandler_PlanActions_Call {
	_c.Call.Return(run)
	return _c
}

// Publish provides a mock function with given fields: ctx
func (_m *MockTaskHandler) Publish(ctx context.Context) {
	_m.Called(ctx)
}

// MockTaskHandler_Publish_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Publish'
type MockTaskHandler_Publish_Call struct {
	*mock.Call
}

// Publish is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockTaskHandler_Expecter) Publish(ctx interface{}) *MockTaskHandler_Publish_Call {
	return &MockTaskHandler_Publish_Call{Call: _e.mock.On("Publish", ctx)}
}

func (_c *MockTaskHandler_Publish_Call) Run(run func(ctx context.Context)) *MockTaskHandler_Publish_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockTaskHandler_Publish_Call) Return() *MockTaskHandler_Publish_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockTaskHandler_Publish_Call) RunAndReturn(run func(context.Context)) *MockTaskHandler_Publish_Call {
	_c.Call.Return(run)
	return _c
}

// Query provides a mock function with given fields: ctx
func (_m *MockTaskHandler) Query(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Query")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockTaskHandler_Query_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Query'
type MockTaskHandler_Query_Call struct {
	*mock.Call
}

// Query is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockTaskHandler_Expecter) Query(ctx interface{}) *MockTaskHandler_Query_Call {
	return &MockTaskHandler_Query_Call{Call: _e.mock.On("Query", ctx)}
}

func (_c *MockTaskHandler_Query_Call) Run(run func(ctx context.Context)) *MockTaskHandler_Query_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockTaskHandler_Query_Call) Return(_a0 error) *MockTaskHandler_Query_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockTaskHandler_Query_Call) RunAndReturn(run func(context.Context) error) *MockTaskHandler_Query_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockTaskHandler creates a new instance of MockTaskHandler. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockTaskHandler(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockTaskHandler {
	mock := &MockTaskHandler{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
