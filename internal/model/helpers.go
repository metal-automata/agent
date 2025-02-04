package model

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/bmc-toolbox/common"
	"github.com/pkg/errors"

	fleetdbapi "github.com/metal-automata/fleetdb/pkg/api/v1"
)

var (
	ErrContextCancelled = errors.New("context canceled")
)

// Sleep, return if context is canceled
func SleepInContext(ctx context.Context, t time.Duration) error {
	// skip sleep in tests
	if os.Getenv(EnvTesting) == "1" {
		return nil
	}

	select {
	case <-time.After(t):
		return nil
	case <-ctx.Done():
		return ErrContextCancelled
	}
}

// FindComponentByNameModel returns a component that matches the name field.
func FindComponentByNameModel(items []*fleetdbapi.ServerComponent, cSlug string, cModels []string) *fleetdbapi.ServerComponent {
	// identify components that match the slug
	slugsMatch := []*fleetdbapi.ServerComponent{}

	for _, component := range items {
		component := component
		// skip non matching component slug
		if !strings.EqualFold(cSlug, component.Name) {
			continue
		}

		// since theres a single BIOS, BMC (:fingers_crossed) component on a machine
		// we look for further and return the found component
		if strings.EqualFold(common.SlugBIOS, cSlug) || strings.EqualFold(common.SlugBMC, cSlug) {
			return component
		}

		slugsMatch = append(slugsMatch, component)
	}

	// none found
	if len(slugsMatch) == 0 {
		return nil
	}

	// multiple components identified, match component by model
	for _, find := range cModels {
		for _, component := range slugsMatch {
			find = strings.ToLower(strings.TrimSpace(find))
			if strings.Contains(strings.ToLower(component.Model), find) {
				return component
			}
		}
	}

	return nil
}
