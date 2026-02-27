package plugincombinedluactxcontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestLuaCtxContractCoreMethods(t *testing.T) {
	testutil.RequirePlugins(t, "plugin-automation", "plugin-test-clean")

	client := http.Client{Timeout: 3 * time.Second}
	nonce := fmt.Sprintf("%d", time.Now().UnixNano())

	autoDeviceID := "automation-ctx-device-" + nonce
	autoEntityID := "automation-ctx-entity-" + nonce
	cleanDeviceID := "clean-ctx-device-" + nonce
	cleanEntityID := "clean-ctx-entity-" + nonce

	createDevice(t, client, "plugin-automation", autoDeviceID)
	createEntity(t, client, "plugin-automation", autoDeviceID, autoEntityID)
	createDevice(t, client, "plugin-test-clean", cleanDeviceID)
	createEntity(t, client, "plugin-test-clean", cleanDeviceID, cleanEntityID)

	scriptPath, statePath := scriptPaths(t, "plugin-automation", autoDeviceID, autoEntityID)
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir script dir failed: %v", err)
	}
	script := fmt.Sprintf(`
function OnInit(Ctx)
  Ctx:OnCommand("plugin-automation.%s.%s.PowerOn", "DoPowerOn")
end

function DoPowerOn(Ctx, Command)
  local devices = Ctx:FindDevices({PluginID="plugin-test-clean", DeviceID="%s", Limit=5})
  Ctx:SetState("device_count", #devices)

  local entities = Ctx:FindEntities({PluginID="plugin-test-clean", DeviceID="%s", EntityID="%s", Limit=5})
  Ctx:SetState("entity_count", #entities)

  local dev, derr = Ctx:GetDevice("plugin-test-clean", "%s")
  if derr == nil and dev ~= nil and dev.DeviceID == "%s" then
    Ctx:SetState("got_device", true)
  end

  local ent, eerr = Ctx:GetEntity("plugin-test-clean", "%s", "%s")
  if eerr == nil and ent ~= nil and ent.EntityID == "%s" then
    Ctx:SetState("got_entity", true)
  end

  local ack, cerr = Ctx:SendCommand({
    PluginID="plugin-test-clean",
    DeviceID="%s",
    EntityID="%s",
    Payload={type="Noop"}
  })
  if cerr == nil and ack ~= nil and ack.CommandID ~= nil then
    Ctx:SetState("command_ok", true)
    Ctx:SetState("command_id", ack.CommandID)
  end

  local ok, ierr = Ctx:EmitEvent({
    DeviceID="%s",
    EntityID="%s",
    Payload={type="script-emit"}
  })
  if ierr == nil and ok == true then
    Ctx:SetState("emit_ok", true)
  end
end
`, autoDeviceID, autoEntityID,
		cleanDeviceID,
		cleanDeviceID, cleanEntityID,
		cleanDeviceID, cleanDeviceID,
		cleanDeviceID, cleanEntityID, cleanEntityID,
		cleanDeviceID, cleanEntityID,
		autoDeviceID, autoEntityID)

	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write script failed: %v", err)
	}
	_ = os.Remove(statePath)

	postCommandWithRetry(t, client, "plugin-automation", autoDeviceID, autoEntityID, map[string]any{"type": "PowerOn"})

	waitForScriptState(t, statePath, func(state map[string]any) bool {
		return intFromState(state, "device_count") == 1 &&
			intFromState(state, "entity_count") == 1 &&
			boolFromState(state, "got_device") &&
			boolFromState(state, "got_entity") &&
			boolFromState(state, "command_ok") &&
			boolFromState(state, "emit_ok") &&
			strFromState(state, "command_id") != ""
	}, 7*time.Second)
}

func createDevice(t *testing.T, client http.Client, pluginID, deviceID string) {
	t.Helper()
	url := fmt.Sprintf("%s/api/plugins/%s/devices", testutil.APIBaseURL(), pluginID)
	dev := types.Device{ID: deviceID, SourceID: "src-" + deviceID, SourceName: deviceID, LocalName: deviceID}
	body, _ := json.Marshal(dev)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("create device failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create device unexpected status=%d plugin=%s id=%s", resp.StatusCode, pluginID, deviceID)
	}
}

func createEntity(t *testing.T, client http.Client, pluginID, deviceID, entityID string) {
	t.Helper()
	url := fmt.Sprintf("%s/api/plugins/%s/devices/%s/entities", testutil.APIBaseURL(), pluginID, deviceID)
	ent := types.Entity{ID: entityID, Domain: "switch", LocalName: entityID}
	body, _ := json.Marshal(ent)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("create entity failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create entity unexpected status=%d plugin=%s device=%s entity=%s", resp.StatusCode, pluginID, deviceID, entityID)
	}
}

func postCommand(t *testing.T, client http.Client, pluginID, deviceID, entityID string, payload map[string]any) {
	t.Helper()
	url := fmt.Sprintf("%s/api/plugins/%s/devices/%s/entities/%s/commands", testutil.APIBaseURL(), pluginID, deviceID, entityID)
	body, _ := json.Marshal(payload)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("post command failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("post command unexpected status: %d", resp.StatusCode)
	}
}

func postCommandWithRetry(t *testing.T, client http.Client, pluginID, deviceID, entityID string, payload map[string]any) {
	t.Helper()
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		status := postCommandStatus(t, client, pluginID, deviceID, entityID, payload)
		if status == http.StatusAccepted {
			return
		}
		if status != http.StatusForbidden && status != http.StatusBadGateway {
			t.Fatalf("post command unexpected status: %d", status)
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("post command did not become accepted within timeout")
}

func postCommandStatus(t *testing.T, client http.Client, pluginID, deviceID, entityID string, payload map[string]any) int {
	t.Helper()
	url := fmt.Sprintf("%s/api/plugins/%s/devices/%s/entities/%s/commands", testutil.APIBaseURL(), pluginID, deviceID, entityID)
	body, _ := json.Marshal(payload)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("post command failed: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func scriptPaths(t *testing.T, pluginID, deviceID, entityID string) (string, string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	path := wd
	for i := 0; i < 8; i++ {
		runtimePath := filepath.Join(path, ".build", "runtime.json")
		if _, err := os.Stat(runtimePath); err == nil {
			base := filepath.Join(path, ".build", "data", pluginID, "devices", deviceID, "entities")
			return filepath.Join(base, entityID+".lua"), filepath.Join(base, entityID+".state.lua.json")
		}
		altRuntimePath := filepath.Join(path, "test", ".build", "runtime.json")
		if _, err := os.Stat(altRuntimePath); err == nil {
			base := filepath.Join(path, "test", ".build", "data", pluginID, "devices", deviceID, "entities")
			return filepath.Join(base, entityID+".lua"), filepath.Join(base, entityID+".state.lua.json")
		}
		next := filepath.Dir(path)
		if next == path {
			break
		}
		path = next
	}
	t.Fatalf("could not locate .build/runtime.json from %s", wd)
	return "", ""
}

func waitForScriptState(t *testing.T, statePath string, predicate func(map[string]any) bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(statePath)
		if err == nil {
			var state map[string]any
			if err := json.Unmarshal(data, &state); err == nil {
				if predicate(state) {
					return
				}
			}
		}
		time.Sleep(120 * time.Millisecond)
	}
	t.Fatalf("script state %s did not satisfy predicate within %s", statePath, timeout)
}

func intFromState(state map[string]any, key string) int {
	raw, ok := state[key].(float64)
	if !ok {
		return 0
	}
	return int(raw)
}

func boolFromState(state map[string]any, key string) bool {
	raw, ok := state[key].(bool)
	return ok && raw
}

func strFromState(state map[string]any, key string) string {
	raw, _ := state[key].(string)
	return raw
}
