package pluginfrigate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestFrigateDiscovery(t *testing.T) {
	testutil.RequirePlugin(t, "plugin-frigate")

	// 1. Mock Frigate API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/config":
			fmt.Fprintln(w, `{"cameras":{"it-camera":{"enabled":true,"name":"IT Camera","detect":{"enabled":true},"record":{"enabled":false}}}}`)
		case "/api/stats":
			fmt.Fprintln(w, `{"cameras":{"it-camera":{"camera_fps":15.0,"process_fps":14.5}}}`)
		case "/api/streams":
			fmt.Fprintln(w, `{"it-camera":{"producers":[{"url":"rtsp://mock/it","remote_addr":"127.0.0.1"}]}}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer ts.Close()

	// 2. Configure Plugin to use Mock API
	// We use the system config entity.
	url := testutil.APIBaseURL() + "/api/plugins/plugin-frigate/devices/frigate-system/entities/frigate-config/commands"
	payload := map[string]any{
		"frigate_url": ts.URL,
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to send config update command: %v", err)
	}
	defer resp.Body.Close()
	// Gateway returns 202 Accepted for commands
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		t.Fatalf("config update command failed with status %d", resp.StatusCode)
	}

	// 3. Verify Discovery via Gateway API
	deviceID := "frigate-device-it-camera"
	entityID := "frigate-entity-it-camera"

	waitForDevice(t, deviceID, 10*time.Second)
	waitForEntity(t, deviceID, entityID, 10*time.Second)
}

func TestSystemDevicePresence(t *testing.T) {
	testutil.RequirePlugin(t, "plugin-frigate")

	deviceID := "frigate-system"
	entityID := "frigate-config"

	// Prove device exists and can be retrieved
	waitForDevice(t, deviceID, 5*time.Second)
	// Prove entity exists under that device and can be retrieved
	waitForEntityMetadata(t, deviceID, entityID, 5*time.Second)
}

func waitForEntityMetadata(t *testing.T, deviceID, expectedID string, timeout time.Duration) {
	t.Helper()
	client := http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	url := testutil.APIBaseURL() + "/api/plugins/plugin-frigate/devices/" + deviceID + "/entities"
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			var entities []types.Entity
			if decodeErr := json.NewDecoder(resp.Body).Decode(&entities); decodeErr == nil {
				resp.Body.Close()
				for _, e := range entities {
					if e.ID == expectedID {
						return
					}
				}
			} else {
				resp.Body.Close()
			}
		} else if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("entity %q not found within %v", expectedID, timeout)
}


func waitForDevice(t *testing.T, expectedID string, timeout time.Duration) {
	t.Helper()
	client := http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	url := testutil.APIBaseURL() + "/api/plugins/plugin-frigate/devices"
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			var devices []types.Device
			if decodeErr := json.NewDecoder(resp.Body).Decode(&devices); decodeErr == nil {
				resp.Body.Close()
				for _, d := range devices {
					if d.ID == expectedID {
						return
					}
				}
			} else {
				resp.Body.Close()
			}
		} else if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("device %q not discovered within %v", expectedID, timeout)
}

func waitForEntity(t *testing.T, deviceID, expectedID string, timeout time.Duration) {
	t.Helper()
	client := http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	url := testutil.APIBaseURL() + "/api/plugins/plugin-frigate/devices/" + deviceID + "/entities"
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			var entities []types.Entity
			if decodeErr := json.NewDecoder(resp.Body).Decode(&entities); decodeErr == nil {
				resp.Body.Close()
				for _, e := range entities {
					if e.ID == expectedID {
						// Also check state
						var state struct {
							StreamURL string `json:"stream_url"`
						}
						json.Unmarshal(e.Data.Reported, &state)
						if state.StreamURL == "rtsp://mock/it" {
							return
						}
					}
				}
			} else {
				resp.Body.Close()
			}
		} else if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("entity %q with correct state not discovered within %v", expectedID, timeout)
}
