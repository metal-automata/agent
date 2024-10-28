package model

import rctypes "github.com/metal-automata/rivets/condition"

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
)

// Returns the Conditions supported by this agent
func ConditionKinds() []rctypes.Kind {
	return []rctypes.Kind{
		rctypes.Inventory,
		rctypes.FirmwareInstall,
	}
}

// AppKinds returns the supported agent app kinds
func AppKinds() []AppKind { return []AppKind{AppKindService, AppKindCLI} }

// StoreKinds returns the supported asset inventory, firmware configuration sources
func StoreKinds() []StoreKind {
	return []StoreKind{InventoryStoreYAML, InventoryStoreServerservice}
}
