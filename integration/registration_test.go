package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/slidebolt/sdk-types"
)

func TestFullMeshDiscovery(t *testing.T) {
	resp, err := http.Get(apiBaseURL() + "/api/plugins")
	if err != nil {
		t.Fatalf("Failed to fetch plugin registry: %v", err)
	}
	defer resp.Body.Close()

	var registry map[string]types.Registration
	json.NewDecoder(resp.Body).Decode(&registry)

	// gateway self-registers + 3 test plugins = 4 minimum
	const minPlugins = 4
	if len(registry) < minPlugins {
		t.Errorf("Expected at least %d registered plugins, found %d", minPlugins, len(registry))
	}
	fmt.Printf("PASS: Registry verified — %d plugins registered.\n", len(registry))
}
