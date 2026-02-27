package plugintestcombineddualdevicecreation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestCreateDeviceOnBothPlugins(t *testing.T) {
	pluginA := "plugin-test-clean"
	pluginB := "plugin-test-slow"
	testutil.RequirePlugins(t, pluginA, pluginB)

	client := http.Client{}
	createAndVerify := func(pluginID, deviceID string) {
		t.Helper()
		url := fmt.Sprintf("%s/api/plugins/%s/devices", testutil.APIBaseURL(), pluginID)
		dev := types.Device{
			ID:         deviceID,
			SourceID:   "src-" + pluginID + "-" + deviceID,
			SourceName: "Source " + deviceID,
			LocalName:  "Local " + deviceID,
		}
		payload, _ := json.Marshal(dev)
		resp, err := client.Post(url, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			t.Fatalf("create device on %s failed: %v", pluginID, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("create device on %s unexpected status: %d", pluginID, resp.StatusCode)
		}

		resp, err = client.Get(url)
		if err != nil {
			t.Fatalf("list devices on %s failed: %v", pluginID, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list devices on %s unexpected status: %d", pluginID, resp.StatusCode)
		}

		var devices []types.Device
		if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
			t.Fatalf("decode devices on %s failed: %v", pluginID, err)
		}

		for _, got := range devices {
			if got.ID == deviceID {
				return
			}
		}
		t.Fatalf("created device %q not found on %s", deviceID, pluginID)
	}

	createAndVerify(pluginA, "combined-clean-dev-1")
	createAndVerify(pluginB, "combined-slow-dev-1")
}
