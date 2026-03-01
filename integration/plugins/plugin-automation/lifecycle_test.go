package pluginautomation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestAutomationLifecycle(t *testing.T) {
	const pluginID = "plugin-automation"
	testutil.RequirePlugin(t, pluginID)

	client := &http.Client{Timeout: 2 * time.Second}

	// 1. Create a Device
	deviceReq := types.Device{
		ID:         "auto-dev-1",
		SourceID:   "source-1",
		SourceName: "Automation Controller",
		LocalName:  "My Rules Engine",
	}
	deviceBody, _ := json.Marshal(deviceReq)
	
	resp, err := client.Post(
		fmt.Sprintf("%s/api/plugins/%s/devices", testutil.APIBaseURL(), pluginID),
		"application/json",
		bytes.NewBuffer(deviceBody),
	)
	if err != nil {
		t.Fatalf("Failed to create device: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 creating device, got %d", resp.StatusCode)
	}

	// 2. Create an Entity on that Device
	entityReq := types.Entity{
		ID:        "rule-1",
		DeviceID:  "auto-dev-1",
		Domain:    "automation",
		LocalName: "Night Mode Rule",
		Actions:   []string{"enable", "disable", "trigger"},
	}
	entityBody, _ := json.Marshal(entityReq)
	
	resp, err = client.Post(
		fmt.Sprintf("%s/api/plugins/%s/devices/%s/entities", testutil.APIBaseURL(), pluginID, "auto-dev-1"),
		"application/json",
		bytes.NewBuffer(entityBody),
	)
	if err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 creating entity, got %d", resp.StatusCode)
	}

	// 3. Verify Device exists in List
	resp, err = client.Get(fmt.Sprintf("%s/api/plugins/%s/devices", testutil.APIBaseURL(), pluginID))
	if err != nil {
		t.Fatalf("Failed to list devices: %v", err)
	}
	defer resp.Body.Close()
	
	var devices []types.Device
	json.NewDecoder(resp.Body).Decode(&devices)
	
	foundDev := false
	for _, d := range devices {
		if d.ID == "auto-dev-1" {
			foundDev = true
			break
		}
	}
	if !foundDev {
		t.Errorf("Device auto-dev-1 not found in list")
	}

	// 4. Verify Entity exists in List
	resp, err = client.Get(fmt.Sprintf("%s/api/plugins/%s/devices/%s/entities", testutil.APIBaseURL(), pluginID, "auto-dev-1"))
	if err != nil {
		t.Fatalf("Failed to list entities: %v", err)
	}
	defer resp.Body.Close()

	var entities []types.Entity
	json.NewDecoder(resp.Body).Decode(&entities)

	foundEnt := false
	for _, e := range entities {
		if e.ID == "rule-1" {
			foundEnt = true
			if e.Domain != "automation" {
				t.Errorf("Expected domain automation, got %s", e.Domain)
			}
			break
		}
	}
	if !foundEnt {
		t.Errorf("Entity rule-1 not found in list")
	}
}
