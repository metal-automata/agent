package model

import (
	rctypes "github.com/metal-automata/rivets/condition"
)

type (
	AppKind   string
	StoreKind string
	// LogLevel is the logging level string.
	LogLevel string
	RunMode  string
)

const (
	AppName                = "agent"
	AppKindService AppKind = "service"
	AppKindCLI     AppKind = "cli"

	RunInband    RunMode = "inband"
	RunOutofband RunMode = "outofband"

	InventoryStoreYAML          StoreKind = "yaml"
	InventoryStoreServerservice StoreKind = "serverservice"

	LogLevelInfo  LogLevel = "info"
	LogLevelDebug LogLevel = "debug"
	LogLevelTrace LogLevel = "trace"

	// envTesting is set by tests to '1' to skip sleeps and backoffs in the handlers.
	//
	// nolint:gosec // no gosec, this isn't a credential
	EnvTesting = "ENV_TESTING"

	// task states
	//
	// states the task state machine transitions through
	StatePending   = rctypes.Pending
	StateActive    = rctypes.Active
	StateSucceeded = rctypes.Succeeded
	StateFailed    = rctypes.Failed

	TaskDataStructVersion = "1.0"
)

// Returns the Conditions supported by this agent
func ConditionKinds() []rctypes.Kind {
	return []rctypes.Kind{
		rctypes.Inventory,
		rctypes.FirmwareInstall,
		rctypes.ServerControl,
	}
}

// AppKinds returns the supported agent app kinds
func AppKinds() []AppKind { return []AppKind{AppKindService, AppKindCLI} }

// StoreKinds returns the supported asset inventory, firmware configuration sources
func StoreKinds() []StoreKind {
	return []StoreKind{InventoryStoreYAML, InventoryStoreServerservice}
}
