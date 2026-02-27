package plugincombinedluaautomationsystemclean

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

func TestLuaEventTickDrivesCrossPluginCommand(t *testing.T) {
	testutil.RequirePlugins(t, "plugin-automation", "plugin-system", "plugin-test-clean")

	client := http.Client{Timeout: 3 * time.Second}
	nonce := fmt.Sprintf("%d", time.Now().UnixNano())

	autoDeviceID := "automation-script-device-" + nonce
	autoEntityID := "party-switch-" + nonce
	cleanDeviceID := "clean-device-" + nonce
	cleanEntityID := "clean-entity-" + nonce

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
  Ctx:OnEvent("plugin-system.tick", "DoTick")
end

function DoTick(Ctx, EventRef)
  local ticks = Ctx:GetState("tick_count")
  if ticks == nil then ticks = 0 end
  ticks = ticks + 1
  Ctx:SetState("tick_count", ticks)

  local existing = Ctx:GetState("last_command_id")
  if existing == nil then
    local ack, err = Ctx:SendCommand("plugin-test-clean", "%s", "%s", {type = "noop"})
    if err == nil and ack ~= nil and ack.CommandID ~= nil then
      Ctx:SetState("last_command_id", ack.CommandID)
    end
  end
end
`, cleanDeviceID, cleanEntityID)
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write script failed: %v", err)
	}
	_ = os.Remove(statePath)

	waitForScriptState(t, statePath, func(state map[string]any) bool {
		rawTicks, ok := state["tick_count"].(float64)
		if !ok || int(rawTicks) < 2 {
			return false
		}
		rawCmd, ok := state["last_command_id"].(string)
		return ok && rawCmd != ""
	}, 12*time.Second)
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
