package pluginesphome

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestESPHomeDiscovery(t *testing.T) {
	const pluginID = "plugin-esphome"
	testutil.RequirePlugin(t, pluginID)

	client := &http.Client{Timeout: 2 * time.Second}
	// List devices
	resp, err := client.Get(testutil.APIBaseURL() + "/api/plugins/" + pluginID + "/devices")
	if err != nil {
		t.Fatalf("failed to list devices: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var devices []types.Device
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		t.Fatalf("failed to decode devices: %v", err)
	}

	// Since we don't have a real dashboard, we might have 0 devices or whatever is in .build/data
	// But the fact that it responded is good.
	t.Logf("ESPHome returned %d devices", len(devices))
}
