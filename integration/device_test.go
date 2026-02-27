package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/slidebolt/sdk-types"
)

func TestDeviceIsolation(t *testing.T) {
	pluginIDs := []struct {
		id       string
		metaWant string
	}{
		{"plugin-test-clean", "clean-metadata"},
		{"plugin-test-slow", "slow-metadata"},
		{"plugin-test-flaky", "flaky-recovery-meta"},
	}

	client := http.Client{}

	fmt.Println("Waiting for plugin-test-flaky to be healthy...")
	if !waitForPlugin("plugin-test-flaky", 5*time.Second) {
		t.Fatal("plugin-test-flaky did not become healthy within timeout")
	}

	for _, p := range pluginIDs {
		fmt.Printf("Creating 25 devices for plugin: %s\n", p.id)
		for i := 1; i <= 25; i++ {
			dev := types.Device{
				ID:         fmt.Sprintf("dev-%d", i),
				SourceID:   fmt.Sprintf("src-%s-%d", p.id, i),
				SourceName: fmt.Sprintf("Source Name %d", i),
				LocalName:  fmt.Sprintf("Local %d", i),
			}
			payload, _ := json.Marshal(dev)
			url := fmt.Sprintf("%s/api/plugins/%s/devices", apiBaseURL(), p.id)
			resp, err := client.Post(url, "application/json", bytes.NewBuffer(payload))
			if err != nil || resp.StatusCode != http.StatusOK {
				t.Fatalf("Failed to create device %d for plugin %s: err=%v status=%v",
					i, p.id, err, statusCode(resp))
			}
			resp.Body.Close()
		}
	}

	for _, p := range pluginIDs {
		url := fmt.Sprintf("%s/api/plugins/%s/devices", apiBaseURL(), p.id)
		resp, err := client.Get(url)
		if err != nil {
			t.Fatalf("Failed to fetch devices for %s: %v", p.id, err)
		}

		var devices []types.Device
		json.NewDecoder(resp.Body).Decode(&devices)
		resp.Body.Close()

		if len(devices) != 25 {
			t.Errorf("Plugin %s: expected 25 devices, got %d", p.id, len(devices))
		}

		for _, dev := range devices {
			if dev.Name() == "" {
				t.Errorf("Plugin %s: device %s has no derived name", p.id, dev.ID)
			}
			if dev.Config.Meta != p.metaWant {
				t.Errorf("Plugin %s: device %s Config.Meta = %q, want %q",
					p.id, dev.ID, dev.Config.Meta, p.metaWant)
			}
		}
		fmt.Printf("PASS: Plugin %s: 25 devices verified (Config.Meta=%q)\n", p.id, p.metaWant)
	}
}

func statusCode(r *http.Response) int {
	if r == nil {
		return -1
	}
	return r.StatusCode
}
