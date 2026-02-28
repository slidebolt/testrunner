package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

// TestDeviceFileWrittenOnCreate verifies the explicit POST /devices path writes
// the device JSON file — this is the happy path and should pass.
func TestDeviceFileWrittenOnCreate(t *testing.T) {
	const pluginID = "plugin-test-clean"
	testutil.RequirePlugin(t, pluginID)

	dataDir := testutil.PluginDataDir(pluginID)
	if dataDir == "" {
		t.Fatal("could not locate plugin data directory")
	}

	body, _ := json.Marshal(types.Device{
		ID:        "test-device-persist",
		SourceID:  "src-001",
		LocalName: "Persistence Test Device",
	})

	resp, err := http.Post(
		testutil.APIBaseURL()+"/api/plugins/"+pluginID+"/devices",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create device request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating device, got %d", resp.StatusCode)
	}

	var created types.Device
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("failed decoding created device: %v", err)
	}
	if created.ID == "" {
		t.Fatal("created device has no ID")
	}

	deviceFile := filepath.Join(dataDir, "devices", created.ID+".json")
	data, err := os.ReadFile(deviceFile)
	if err != nil {
		t.Fatalf("device JSON file not found at %s: %v", deviceFile, err)
	}

	var persisted types.Device
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("device file contains invalid JSON: %v", err)
	}
	if persisted.ID != created.ID {
		t.Errorf("persisted device ID %q does not match created ID %q", persisted.ID, created.ID)
	}

	fmt.Printf("PASS: device file written at %s\n", deviceFile)
}

// TestDeviceFileWrittenWhenEntityCreated proves the bug: creating an entity for a
// device that was never explicitly registered via POST /devices should still
// result in a device JSON file being written. Currently it does not — the entity
// file is created (saveEntity is called) but saveDevice is never called, so the
// device is invisible to GET /devices.
func TestDeviceFileWrittenWhenEntityCreated(t *testing.T) {
	const pluginID = "plugin-test-clean"
	const deviceID = "implicit-device-001"
	testutil.RequirePlugin(t, pluginID)

	dataDir := testutil.PluginDataDir(pluginID)
	if dataDir == "" {
		t.Fatal("could not locate plugin data directory")
	}

	// Create an entity directly for a device that has never been registered.
	// This calls entities/create on the runner → saveEntity is called → entity
	// file written. But saveDevice is never called.
	body, _ := json.Marshal(types.Entity{
		ID:        "implicit-entity-001",
		Domain:    "switch",
		LocalName: "Implicit Entity",
	})
	resp, err := http.Post(
		testutil.APIBaseURL()+"/api/plugins/"+pluginID+"/devices/"+deviceID+"/entities",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create entity request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating entity, got %d", resp.StatusCode)
	}

	// Entity file should exist.
	entityFile := filepath.Join(dataDir, "devices", deviceID, "entities", "implicit-entity-001.json")
	if _, err := os.Stat(entityFile); err != nil {
		t.Fatalf("entity file missing at %s — prerequisite for this test: %v", entityFile, err)
	}

	// Device file must also exist. Without it, the device is invisible to
	// GET /devices (loadDevices only globs devices/*.json) and any restart
	// will lose the context needed to associate entities with their device.
	deviceFile := filepath.Join(dataDir, "devices", deviceID+".json")
	data, err := os.ReadFile(deviceFile)
	if err != nil {
		t.Fatalf("BUG: entity file exists but device JSON file is missing at %s", deviceFile)
	}

	var persisted types.Device
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("device file contains invalid JSON: %v", err)
	}
	if persisted.ID != deviceID {
		t.Errorf("persisted device ID %q != expected %q", persisted.ID, deviceID)
	}

	fmt.Printf("PASS: device file written at %s\n", deviceFile)
}
