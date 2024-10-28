package cmd

import (
	"context"
	"log"

	"github.com/equinix-labs/otel-init-go/otelinit"
	"github.com/google/uuid"
	"github.com/metal-automata/agent/internal/app"
	"github.com/metal-automata/agent/internal/ctrl"
	"github.com/metal-automata/agent/internal/metrics"
	"github.com/metal-automata/agent/internal/model"
	"github.com/metal-automata/agent/internal/service"
	"github.com/metal-automata/agent/internal/store"

	rctypes "github.com/metal-automata/rivets/condition"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	// nolint:gosec // profiling endpoint listens on localhost.
	_ "net/http/pprof"
)

var cmdRun = &cobra.Command{
	Use:   "service",
	Short: "Runs Agent service to listen for events and execute on tasks",
	Run: func(cmd *cobra.Command, _ []string) {
		var mode model.RunMode
		if runsInband {
			mode = model.RunInband
		} else {
			mode = model.RunOutofband
		}

		runHandler(cmd.Context(), mode)
	},
}

// run worker command
var (
	dryrun         bool
	runsInband     bool
	runsOutofband  bool
	faultInjection bool
	facilityCode   string
	storeKind      string
	inbandServerID string
)

var (
	ErrInventoryStore = errors.New("inventory store error")
)

func runHandler(ctx context.Context, mode model.RunMode) {
	agent, termCh, err := app.New(
		model.AppKindService,
		model.StoreKind(storeKind),
		cfgFile,
		logLevel,
		enableProfiling,
		mode,
	)
	if err != nil {
		log.Fatal(err)
	}

	// serve metrics endpoint
	metrics.ListenAndServe()

	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, "agent-"+string(mode))
	defer otelShutdown(ctx)

	// Setup cancel context with cancel func.
	ctx, cancelFunc := context.WithCancel(ctx)

	// routine listens for termination signal and cancels the context
	go func() {
		<-termCh
		agent.Logger.Info("got TERM signal, exiting...")
		cancelFunc()
	}()

	repository, err := initStore(ctx, agent.Config, agent.Logger)
	if err != nil {
		agent.Logger.Fatal(err)
	}

	if facilityCode == "" {
		agent.Logger.Fatal("--facility-code parameter required")
	}

	switch mode {
	case model.RunInband:
		runInband(ctx, agent, repository)
		return
	case model.RunOutofband:
		runOutofband(ctx, agent, repository)
		return
	default:
		agent.Logger.Fatal("unsupported run mode: " + mode)
	}
}

func runOutofband(ctx context.Context, agent *app.App, repository store.Repository) {
	natsCfg, err := agent.NatsParams()
	if err != nil {
		agent.Logger.Fatal(err)
	}

	nc := ctrl.NewNatsController(
		model.AppName,
		facilityCode,
		natsCfg.NatsURL,
		natsCfg.CredsFile,
		model.ConditionKinds(),
		ctrl.WithConcurrency(agent.Config.Concurrency),
		ctrl.WithKVReplicas(natsCfg.KVReplicas),
		ctrl.WithLogger(agent.Logger),
		ctrl.WithConnectionTimeout(natsCfg.ConnectTimeout),
	)

	if err := nc.Connect(ctx); err != nil {
		agent.Logger.Fatal(err)
	}

	service.RunOutofband(
		ctx,
		dryrun,
		faultInjection,
		repository,
		nc,
		agent.Logger,
	)
}

func runInband(ctx context.Context, agent *app.App, repository store.Repository) {
	cfgOrcAPI := agent.Config.OrchestratorAPIParams
	orcConfig := &ctrl.OrchestratorAPIConfig{
		Endpoint:             cfgOrcAPI.Endpoint,
		AuthDisabled:         cfgOrcAPI.AuthDisabled,
		OidcIssuerEndpoint:   cfgOrcAPI.OidcIssuerEndpoint,
		OidcAudienceEndpoint: cfgOrcAPI.OidcAudienceEndpoint,
		OidcClientSecret:     cfgOrcAPI.OidcClientSecret,
		OidcClientID:         cfgOrcAPI.OidcClientID,
		OidcClientScopes:     cfgOrcAPI.OidcClientScopes,
	}

	nc, err := ctrl.NewHTTPController( //nolint:contextcheck // oauth init has its own context
		"agent-inband",
		facilityCode,
		uuid.MustParse(agent.Config.ServerID),
		rctypes.FirmwareInstallInband,
		orcConfig,
		ctrl.WithNATSHTTPLogger(agent.Logger),
	)
	if err != nil {
		agent.Logger.Fatal(err)
	}

	service.RunInband(
		ctx,
		dryrun,
		faultInjection,
		facilityCode,
		repository,
		nc,
		agent.Logger,
	)
}

func initStore(ctx context.Context, config *app.Configuration, logger *logrus.Logger) (store.Repository, error) {
	if storeKind == string(model.InventoryStoreServerservice) {
		return store.NewServerserviceStore(ctx, config.FleetDBAPIOptions, logger)
	}

	return nil, errors.Wrap(ErrInventoryStore, "expected a valid inventory store parameter")
}

func init() {
	cmdRun.PersistentFlags().StringVar(&storeKind, "store", "", "Inventory store to lookup devices for update - fleetdb.")
	cmdRun.PersistentFlags().StringVar(&inbandServerID, "server-id", "", "ServerID when running inband")
	cmdRun.PersistentFlags().BoolVarP(&dryrun, "dry-run", "", false, "In dryrun mode, the agent actions the task without installing firmware")
	cmdRun.PersistentFlags().BoolVarP(&runsInband, "inband", "", false, "Runs agent service in inband firmware mode (expects to run on the target device)")
	cmdRun.PersistentFlags().BoolVarP(&runsOutofband, "outofband", "", false, "Runs service in out-of-band mode (target host is remote)")
	cmdRun.PersistentFlags().BoolVarP(&faultInjection, "fault-injection", "", false, "Tasks can include a Fault attribute to allow fault injection for development purposes")
	cmdRun.PersistentFlags().StringVar(&facilityCode, "facility-code", "", "The facility code this agent instance is associated with")

	if err := cmdRun.MarkPersistentFlagRequired("store"); err != nil {
		log.Fatal(err)
	}

	cmdRun.MarkFlagsMutuallyExclusive("inband", "outofband")
	cmdRun.MarkFlagsOneRequired("inband", "outofband")

	rootCmd.AddCommand(cmdRun)
}
