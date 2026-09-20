// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ruanklein/fmgo"
)

const nativeSetupTimeout = 30 * time.Second

var nativeClient *fmgo.Client

func TestMain(m *testing.M) {
	if os.Getenv("FMGO_HELPER") == "1" {
		os.Exit(m.Run())
	}
	if os.Getenv("FMGO_INTEGRATION") != "1" {
		os.Exit(m.Run())
	}

	client, err := fmgo.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fmgo integration setup: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), nativeSetupTimeout)
	defer cancel()
	if err := validateNativeEnvironment(ctx, client); err != nil {
		fmt.Fprintf(os.Stderr, "fmgo integration setup: %v\n", err)
		os.Exit(1)
	}

	nativeClient = client
	os.Exit(m.Run())
}

func validateNativeEnvironment(ctx context.Context, client *fmgo.Client) error {
	license, err := client.License(ctx)
	if err != nil {
		return fmt.Errorf("check license: %w", err)
	}
	if !license.Accepted {
		return fmt.Errorf("Foundation Models license is not accepted")
	}

	availability, err := client.Available(ctx)
	if err != nil {
		return fmt.Errorf("check model availability: %w", err)
	}
	for _, model := range availability.Models {
		if model.Model == fmgo.ModelSystem && model.Available {
			return nil
		}
	}
	return fmt.Errorf("system Foundation Model is unavailable")
}
