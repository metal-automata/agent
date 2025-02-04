package store

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bmc-toolbox/common"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/metal-automata/agent/internal/app"
	"github.com/metal-automata/agent/internal/metrics"
	"github.com/metal-automata/agent/internal/model"
	"github.com/pkg/errors"

	fleetdbapi "github.com/metal-automata/fleetdb/pkg/api/v1"
	rctypes "github.com/metal-automata/rivets/condition"
	rfleetdb "github.com/metal-automata/rivets/fleetdb"
)

const (
	// connectionTimeout is the maximum amount of time spent on each http connection to fleetdb API.
	connectionTimeout = 30 * time.Second

	pkgName = "internal/store"
)

var (
	ErrNoAttributes          = errors.New("no agent attribute found")
	ErrAttributeList         = errors.New("error in fleetdb API attribute list")
	ErrAttributeCreate       = errors.New("error in fleetdb API attribute create")
	ErrAttributeUpdate       = errors.New("error in fleetdb API attribute update")
	ErrVendorModelAttributes = errors.New("device vendor, model attributes not found in fleetdb API")
	ErrDeviceStatus          = errors.New("error fleetdb API device status")

	ErrDeviceID = errors.New("device UUID error")

	// ErrBMCAddress is returned when an error occurs in the BMC address lookup.
	ErrBMCAddress = errors.New("error in server BMC Address")

	// ErrDeviceState is returned when an error occurs in the device state  lookup.
	ErrDeviceState = errors.New("error in device state")

	// ErrServerserviceAttrObj is retuned when an error occurred in unpacking the attribute.
	ErrServerserviceAttrObj = errors.New("fleetdb API attribute error")

	// ErrServerserviceVersionedAttrObj is retuned when an error occurred in unpacking the versioned attribute.
	ErrServerserviceVersionedAttrObj = errors.New("fleetdb API versioned attribute error")

	// ErrServerserviceQuery is returned when a server service query fails.
	ErrServerserviceQuery = errors.New("fleetdb API query returned error")

	ErrFirmwareSetLookup = errors.New("firmware set error")
)

type FleetDBAPI struct {
	config *app.FleetDBAPIOptions
	// componentSlugs map[string]string
	slugMap rctypes.ComponentSlugMap
	client  *fleetdbapi.Client
	logger  *logrus.Logger
}

func NewServerserviceStore(ctx context.Context, config *app.FleetDBAPIOptions, logger *logrus.Logger) (Repository, error) {
	var client *fleetdbapi.Client
	var err error

	if !config.DisableOAuth {
		client, err = newClientWithOAuth(ctx, config, logger)
		if err != nil {
			return nil, err
		}
	} else {
		client, err = fleetdbapi.NewClientWithToken("fake", config.Endpoint, nil)
		if err != nil {
			return nil, err
		}
	}

	apiclient := &FleetDBAPI{
		client:  client,
		config:  config,
		logger:  logger,
		slugMap: make(rctypes.ComponentSlugMap),
	}

	// add component types if they don't exist
	if err := apiclient.createServerComponentTypes(ctx); err != nil {
		return nil, err
	}

	return apiclient, nil
}

// returns a fleetdb API retryable http client with Otel and Oauth wrapped in
func newClientWithOAuth(ctx context.Context, cfg *app.FleetDBAPIOptions, logger *logrus.Logger) (*fleetdbapi.Client, error) {
	// init retryable http client
	retryableClient := retryablehttp.NewClient()

	// set retryable HTTP client to be the otel http client to collect telemetry
	retryableClient.HTTPClient = otelhttp.DefaultClient

	// disable default debug logging on the retryable client
	if logger.Level < logrus.DebugLevel {
		retryableClient.Logger = nil
	} else {
		retryableClient.Logger = logger
	}

	// setup oidc provider
	provider, err := oidc.NewProvider(ctx, cfg.OidcIssuerEndpoint)
	if err != nil {
		return nil, err
	}

	clientID := "agent"

	if cfg.OidcClientID != "" {
		clientID = cfg.OidcClientID
	}

	// setup oauth configuration
	oauthConfig := clientcredentials.Config{
		ClientID:       clientID,
		ClientSecret:   cfg.OidcClientSecret,
		TokenURL:       provider.Endpoint().TokenURL,
		Scopes:         cfg.OidcClientScopes,
		EndpointParams: url.Values{"audience": []string{cfg.OidcAudienceEndpoint}},
	}

	// wrap OAuth transport, cookie jar in the retryable client
	oAuthclient := oauthConfig.Client(ctx)

	retryableClient.HTTPClient.Transport = oAuthclient.Transport
	retryableClient.HTTPClient.Jar = oAuthclient.Jar

	httpClient := retryableClient.StandardClient()
	httpClient.Timeout = connectionTimeout

	return fleetdbapi.NewClientWithToken(
		cfg.OidcClientSecret,
		cfg.Endpoint,
		httpClient,
	)
}

// Converts from the common.Device to the fleetdbapi.Server type
func (s *FleetDBAPI) ConvertCommonDevice(serverID uuid.UUID, hw *common.Device, collectionMethod model.InstallMethod, checkComponentSlug bool) (*rctypes.Server, error) {
	converter := fleetdbapi.NewComponentConverter(fleetdbapi.CollectionMethod(collectionMethod), s.slugMap, !checkComponentSlug)
	return converter.FromCommonDevice(serverID, hw)
}

func (s *FleetDBAPI) registerErrorMetric(queryKind string) {
	metrics.StoreQueryErrorCount.With(
		prometheus.Labels{
			"storeKind": "serverservice",
			"queryKind": queryKind,
		},
	).Inc()
}

// AssetByID returns an Asset object with various attributes populated.
func (s *FleetDBAPI) AssetByID(ctx context.Context, id string) (*fleetdbapi.Server, error) {
	ctx, span := otel.Tracer(pkgName).Start(ctx, "FleetDBAPI.AssetByID")
	defer span.End()

	deviceUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.Wrap(ErrDeviceID, err.Error()+id)
	}

	params := &fleetdbapi.ServerQueryParams{
		IncludeBMC:        true,
		IncludeComponents: true,
		ComponentParams: &fleetdbapi.ServerComponentGetParams{
			InstalledFirmware: true,
			Status:            true,
			Capabilities:      true,
			Metadata:          []string{fleetdbapi.ComponentMetadataGenericNS},
		},
	}

	// query the server object
	srv, _, err := s.client.GetServer(ctx, deviceUUID, params)
	if err != nil {
		s.registerErrorMetric("GetServer")

		return nil, errors.Wrap(ErrServerserviceQuery, "GetServer: "+err.Error())
	}

	// query credentials
	credential, _, err := s.client.GetCredential(ctx, deviceUUID, fleetdbapi.ServerCredentialTypeBMC)
	if err != nil {
		s.registerErrorMetric("GetCredential")

		return nil, errors.Wrap(ErrServerserviceQuery, "GetCredential: "+err.Error())
	}

	srv.BMC.Username = credential.Username
	srv.BMC.Password = credential.Password

	return srv, nil
}

// FirmwareSetByID returns a list of firmwares part of a firmware set identified by the given id.
func (s *FleetDBAPI) FirmwareSetByID(ctx context.Context, id uuid.UUID) ([]*rctypes.Firmware, error) {
	ctx, span := otel.Tracer(pkgName).Start(ctx, "FleetDBAPI.FirmwareSetByID")
	defer span.End()

	firmwareset, _, err := s.client.GetServerComponentFirmwareSet(ctx, id)
	if err != nil {
		s.registerErrorMetric("GetFirmwareSet")

		return nil, errors.Wrap(ErrServerserviceQuery, "GetFirmwareSet: "+err.Error())
	}

	return intoFirmwaresSlice(firmwareset.ComponentFirmware), nil
}

// FirmwareByDeviceVendorModel returns the firmware for the device vendor, model.
func (s *FleetDBAPI) FirmwareByDeviceVendorModel(ctx context.Context, deviceVendor, deviceModel string) ([]*rctypes.Firmware, error) {
	// lookup agent task attribute
	params := &fleetdbapi.ComponentFirmwareSetListParams{
		AttributeListParams: []fleetdbapi.AttributeListParams{
			{
				Namespace: rfleetdb.FirmwareSetAttributeNS,
				Keys:      []string{"model"},
				Operator:  "eq",
				Value:     deviceModel,
			},
			{
				Namespace: rfleetdb.FirmwareSetAttributeNS,
				Keys:      []string{"vendor"},
				Operator:  "eq",
				Value:     deviceVendor,
			},
		},
	}

	firmwaresets, _, err := s.client.ListServerComponentFirmwareSet(ctx, params)
	if err != nil {
		return nil, errors.Wrap(ErrServerserviceQuery, err.Error())
	}

	if len(firmwaresets) == 0 {
		return nil, errors.Wrap(
			ErrFirmwareSetLookup,
			fmt.Sprintf(
				"lookup by device vendor: %s, model: %s returned no firmware set",
				deviceVendor,
				deviceModel,
			),
		)
	}

	if len(firmwaresets) > 1 {
		return nil, errors.Wrap(
			ErrFirmwareSetLookup,
			fmt.Sprintf(
				"lookup by device vendor: %s, model: %s returned multiple firmware sets, expected one",
				deviceVendor,
				deviceModel,
			),
		)
	}

	if len(firmwaresets[0].ComponentFirmware) == 0 {
		return nil, errors.Wrap(
			ErrFirmwareSetLookup,
			fmt.Sprintf(
				"lookup by device vendor: %s, model: %s returned firmware set with no component firmware",
				deviceVendor,
				deviceModel,
			),
		)
	}

	found := []*rctypes.Firmware{}

	// nolint:gocritic // rangeValCopy - the data is returned by fleetdb API in this form.
	for _, set := range firmwaresets {
		found = append(found, intoFirmwaresSlice(set.ComponentFirmware)...)
	}

	return found, nil
}

func intoFirmwaresSlice(componentFirmware []fleetdbapi.ComponentFirmwareVersion) []*rctypes.Firmware {
	strSliceToLower := func(sl []string) []string {
		lowered := make([]string, 0, len(sl))

		for _, s := range sl {
			lowered = append(lowered, strings.ToLower(s))
		}

		return lowered
	}

	firmwares := make([]*rctypes.Firmware, 0, len(componentFirmware))

	booleanIsTrue := func(b *bool) bool {
		if b != nil && *b {
			return true
		}

		return false
	}

	// nolint:gocritic // rangeValCopy - componentFirmware is returned by fleetdb API in this form.
	for _, firmware := range componentFirmware {
		fw := &rctypes.Firmware{
			ID:            firmware.UUID.String(),
			Vendor:        strings.ToLower(firmware.Vendor),
			Models:        strSliceToLower(firmware.Model),
			FileName:      firmware.Filename,
			Version:       firmware.Version,
			Component:     strings.ToLower(firmware.Component),
			Checksum:      firmware.Checksum,
			URL:           firmware.RepositoryURL,
			InstallInband: *firmware.InstallInband,
			Oem:           *firmware.OEM,
		}

		if booleanIsTrue(firmware.InstallInband) {
			fw.InstallInband = true
		}

		if booleanIsTrue(firmware.OEM) {
			fw.Oem = true
		}

		firmwares = append(firmwares, fw)
	}

	return firmwares
}

func (s *FleetDBAPI) createServerComponentTypes(ctx context.Context) error {
	ctx, span := otel.Tracer(pkgName).Start(ctx, "fleetdbapi.createServerComponentTypes")
	defer span.End()

	existing, err := s.listServerComponentTypes(ctx)
	if err != nil {
		return err
	}

	if len(existing) > 0 {
		return nil
	}

	componentSlugs := []string{
		common.SlugBackplaneExpander,
		common.SlugChassis,
		common.SlugTPM,
		common.SlugGPU,
		common.SlugCPU,
		common.SlugPhysicalMem,
		common.SlugStorageController,
		common.SlugBMC,
		common.SlugBIOS,
		common.SlugDrive,
		common.SlugDriveTypePCIeNVMEeSSD,
		common.SlugDriveTypeSATASSD,
		common.SlugDriveTypeSATAHDD,
		common.SlugNIC,
		common.SlugNICPort,
		common.SlugPSU,
		common.SlugCPLD,
		common.SlugEnclosure,
		common.SlugUnknown,
		common.SlugMainboard,
	}

	for _, slug := range componentSlugs {
		sct := fleetdbapi.ServerComponentType{
			Name: slug,
			Slug: strings.ToLower(slug),
		}

		_, err := s.client.CreateServerComponentType(ctx, sct)
		if err != nil {
			s.registerErrorMetric("CreateServerComponentTypes")

			return errors.Wrap(ErrServerserviceQuery, "CreateServerComponentTypes: "+err.Error())
		}
	}

	return nil
}

func (s *FleetDBAPI) listServerComponentTypes(ctx context.Context) (fleetdbapi.ServerComponentTypeSlice, error) {
	existing, _, err := s.client.ListServerComponentTypes(ctx, nil)
	if err != nil {
		s.registerErrorMetric("ListServerComponentTypes")

		return nil, errors.Wrap(ErrServerserviceQuery, "ListServerComponentTypes: "+err.Error())
	}

	// update cached records
	for _, ct := range existing {
		s.slugMap[ct.Slug] = ct
	}

	return existing, nil
}

func (s *FleetDBAPI) SetComponentInventory(ctx context.Context, serverID uuid.UUID, device *common.Device, method model.CollectionMethod) error {
	currentInventory, err := s.AssetByID(ctx, serverID.String())
	if err != nil {
		return err
	}

	newInventory, err := s.ConvertCommonDevice(serverID, device, model.InstallMethodOutofband, true)
	if err != nil {
		return err
	}

	// initialize component records
	if len(currentInventory.Components) == 0 {
		if _, err := s.client.InitComponentCollection(ctx, serverID, newInventory.Components, fleetdbapi.CollectionMethod(method)); err != nil {
			return errors.Wrap(ErrServerserviceQuery, "InitComponentCollection: "+err.Error())
		}

		s.logger.WithFields(
			logrus.Fields{
				"Server":  serverID.String(),
				"creates": len(newInventory.Components),
			},
		).Info("Component inventory initialized")

		return nil
	}

	// propose component changes
	// - component attribute updates can be published
	// - component additions or removal are to be proposed (which then requires the admin to approve)
	componentsInRecord := fleetdbapi.ServerComponentSlice(currentInventory.Components).AsMap()
	componentsFound := fleetdbapi.ServerComponentSlice(newInventory.Components).AsMap()

	creates := make(fleetdbapi.ServerComponentSlice, 0)
	updates := make(fleetdbapi.ServerComponentSlice, 0)
	deletes := make(fleetdbapi.ServerComponentSlice, 0)

	for key, found := range componentsFound {
		record, exists := componentsInRecord[key]
		if !exists {
			creates = append(creates, found)
			s.logger.WithFields(
				logrus.Fields{
					"Server":      serverID.String(),
					"slug:serial": key,
				},
			).Info("component create")
		} else {
			// update record
			differences, equal := record.Equals(found)
			if !equal {
				updates = append(updates, found)
				s.logger.WithFields(
					logrus.Fields{
						"Server":      serverID.String(),
						"ID":          record.UUID,
						"slug:serial": key,
						"differs":     differences,
					},
				).Info("component update")
			}
		}

		// delete component from map
		delete(componentsInRecord, key)
	}

	// any remaining items in record are to be purged
	// -TBD based on OOB/Inband
	for key, component := range componentsInRecord {
		deletes = append(deletes, component)
		s.logger.WithFields(
			logrus.Fields{
				"Server":      serverID.String(),
				"ID":          component.UUID,
				"slug:serial": key,
			},
		).Info("component delete")
	}

	if len(updates) > 0 {
		if _, err := s.client.UpdateComponentCollection(ctx, serverID, updates, fleetdbapi.CollectionMethod(method)); err != nil {
			return errors.Wrap(ErrServerserviceQuery, "UpdateComponentCollection: "+err.Error())
		}

		s.logger.WithFields(
			logrus.Fields{
				"Server":  serverID.String(),
				"updates": len(updates),
			},
		).Info("Component updates published")

	}

	if len(creates) > 0 || len(deletes) > 0 {
		report := &fleetdbapi.ComponentChangeReport{
			CollectionMethod: string(method),
			Creates:          creates,
			Deletes:          deletes,
		}

		reportResponse, _, err := s.client.ReportComponentChanges(ctx, serverID.String(), report)
		if err != nil {
			return errors.Wrap(ErrServerserviceQuery, "ReportComponentChanges: "+err.Error())
		}

		s.logger.WithFields(
			logrus.Fields{
				"Server":         serverID.String(),
				"changeReportID": reportResponse.ReportID,
				"deletes":        len(deletes),
				"creates":        len(creates),
			},
		).Info("Component change report published")
	}

	if len(creates) == 0 && len(updates) == 0 && len(deletes) == 0 {
		s.logger.WithFields(
			logrus.Fields{
				"Server": serverID.String(),
			},
		).Info("No changes to be published")
	}

	return nil
}
