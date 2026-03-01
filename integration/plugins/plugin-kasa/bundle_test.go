package plugin_kasa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	entityswitch "github.com/slidebolt/sdk-entities/switch"
	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestKasaPlugin(t *testing.T) {
	pluginID := "plugin-kasa"
	testutil.RequirePlugin(t, pluginID)

	t.Run("Discovery and Registration", func(t *testing.T) {
		registry, err := testutil.RegisteredPlugins()
		if err != nil {
			t.Fatalf("failed to get registered plugins: %v", err)
		}
		if _, ok := registry[pluginID]; !ok {
			t.Errorf("plugin %s not found in registry", pluginID)
		}
	})

	t.Run("Device List", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/plugins/%s/devices", testutil.APIBaseURL(), pluginID)
		resp, err := (&http.Client{Timeout: 2 * time.Second}).Get(url)
		if err != nil {
			t.Fatalf("failed to list devices: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var devices []types.Device
		if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
			t.Fatalf("failed to decode devices: %v", err)
		}

		// Since we don't know what real devices are on the network,
		// we can't assert a count, but we can verify the response structure.
		t.Logf("Found %d Kasa devices", len(devices))
	})
}

func TestKasaCommands(t *testing.T) {
	pluginID := "plugin-kasa"
	testutil.RequirePlugin(t, pluginID)

	// In a real integration test environment, we might have a mock device
	// or a specific test device IP configured via environment variables.
	testIP := testutil.PluginEnv(pluginID, "KASA_TEST_DEVICE_IP")
	testMAC := testutil.PluginEnv(pluginID, "KASA_TEST_DEVICE_MAC")

	if testIP == "" || testMAC == "" {
		t.Skip("KASA_TEST_DEVICE_IP or KASA_TEST_DEVICE_MAC not set; skipping command tests")
	}

	deviceID := testMAC
	entityID := "power" // Assuming a switch for the test device

	t.Run("Switch Toggle", func(t *testing.T) {
		// 1. Ensure device and entity exist (Gateway/Plugin should auto-discover if IP is known)
		// We'll try to create the device if it doesn't exist to ensure it's in the system.
		createDevice(t, pluginID, deviceID, testMAC, testIP)

		// 2. Send Turn On Command
		cmdPayload, _ := json.Marshal(entityswitch.Command{Type: entityswitch.ActionTurnOn})
		status := sendCommand(t, pluginID, deviceID, entityID, cmdPayload)

		if status.State != types.CommandSucceeded && status.State != types.CommandPending {
			t.Errorf("expected command success or pending, got %s", status.State)
		}

		// 3. Wait for state to reflect in Gateway
		waitForState(t, pluginID, deviceID, entityID, true)

		// 4. Send Turn Off Command
		cmdPayload, _ = json.Marshal(entityswitch.Command{Type: entityswitch.ActionTurnOff})
		sendCommand(t, pluginID, deviceID, entityID, cmdPayload)
		waitForState(t, pluginID, deviceID, entityID, false)
	})
}

func createDevice(t *testing.T, pluginID, deviceID, mac, ip string) {
	dev := types.Device{
		ID:       deviceID,
		SourceID: mac,
	}
	url := fmt.Sprintf("%s/api/plugins/%s/devices", testutil.APIBaseURL(), pluginID)
	body, _ := json.Marshal(dev)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}
	defer resp.Body.Close()
}

func sendCommand(t *testing.T, pluginID, deviceID, entityID string, payload []byte) types.CommandStatus {
	url := fmt.Sprintf("%s/api/plugins/%s/devices/%s/entities/%s/commands", testutil.APIBaseURL(), pluginID, deviceID, entityID)
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to send command: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}

	var status types.CommandStatus
	json.NewDecoder(resp.Body).Decode(&status)
	return status
}

func waitForState(t *testing.T, pluginID, deviceID, entityID string, expectedPower bool) {
	deadline := time.Now().Add(10 * time.Second)
	url := fmt.Sprintf("%s/api/plugins/%s/devices/%s/entities", testutil.APIBaseURL(), pluginID, deviceID)

	for time.Now().Before(deadline) {
		resp, err := (&http.Client{Timeout: 2 * time.Second}).Get(url)
		if err == nil {
			var entities []types.Entity
			json.NewDecoder(resp.Body).Decode(&entities)
			resp.Body.Close()

			for _, ent := range entities {
				if ent.ID == entityID {
					var state entityswitch.State
					json.Unmarshal(ent.Data.Reported, &state)
					if state.Power == expectedPower {
						return
					}
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Errorf("timed out waiting for state power=%v", expectedPower)
}