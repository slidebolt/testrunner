package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestFullMeshDiscovery(t *testing.T) {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(testutil.APIBaseURL() + "/api/plugins")
	if err != nil {
		t.Fatalf("Failed to fetch plugin registry: %v", err)
	}
	defer resp.Body.Close()

	var registry map[string]types.Registration
	json.NewDecoder(resp.Body).Decode(&registry)

	// Only gateway is mandatory for this harness.
	if _, ok := registry["gateway"]; !ok {
		t.Fatalf("expected gateway to be registered, registry=%v", registry)
	}
	fmt.Printf("PASS: Registry verified — %d plugins registered.\n", len(registry))
}
