package pluginesphome

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestESPHomeRealDiscovery(t *testing.T) {
	const pluginID = "plugin-esphome"
	testutil.RequirePlugin(t, pluginID)

	url := testutil.PluginEnv(pluginID, "ESPHOME_DASHBOARD_URL", "ESPHOME_URL")
	if url == "" {
		t.Skip("no ESPHome dashboard URL configured; skipping real discovery test")
	}

	client := http.Client{Timeout: 3 * time.Second}
	deadline := time.Now().Add(25 * time.Second)
	max := 0
	for time.Now().Before(deadline) {
		resp, err := client.Get(testutil.APIBaseURL() + "/api/plugins/" + pluginID + "/devices")
		if err == nil && resp.StatusCode == http.StatusOK {
			var devices []types.Device
			if err := json.NewDecoder(resp.Body).Decode(&devices); err == nil {
				if len(devices) > max {
					max = len(devices)
				}
			}
			resp.Body.Close()
		} else if resp != nil {
			resp.Body.Close()
		}
		if max > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if max == 0 {
		t.Fatalf("real ESPHome discovery returned 0 devices within timeout (dashboard configured)")
	}

	t.Logf("ESPHome real discovery found %d devices", max)
}
