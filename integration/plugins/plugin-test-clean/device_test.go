package plugintestclean

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

const pluginID = "plugin-test-clean"

func TestDeviceCreateAndMetadata(t *testing.T) {
	testutil.RequirePlugin(t, pluginID)

	client := http.Client{Timeout: 2 * time.Second}
	deviceID := "clean-dev-1"
	url := fmt.Sprintf("%s/api/plugins/%s/devices", testutil.APIBaseURL(), pluginID)

	dev := types.Device{
		ID:         deviceID,
		SourceID:   "src-clean-1",
		SourceName: "Clean Source 1",
		LocalName:  "Clean Local 1",
	}
	payload, _ := json.Marshal(dev)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatalf("create device failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create device unexpected status: %d", resp.StatusCode)
	}

	resp, err = client.Get(url)
	if err != nil {
		t.Fatalf("list devices failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list devices unexpected status: %d", resp.StatusCode)
	}

	var devices []types.Device
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		t.Fatalf("decode devices failed: %v", err)
	}

	for _, got := range devices {
		if got.ID == deviceID {
			return
		}
	}
	t.Fatalf("created device %q not found", deviceID)
}
