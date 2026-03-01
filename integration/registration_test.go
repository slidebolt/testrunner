package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestFullMeshDiscovery(t *testing.T) {
	var registry map[string]types.Registration
	var err error

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		registry, err = testutil.RegisteredPlugins()
		if err == nil && len(registry) > 1 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("Failed to fetch plugin registry: %v", err)
	}

	// Only gateway is mandatory for this harness.
	if _, ok := registry["gateway"]; !ok {
		t.Fatalf("expected gateway to be registered, registry=%v", registry)
	}
	fmt.Printf("PASS: Registry verified — %d plugins registered.\n", len(registry))
}
