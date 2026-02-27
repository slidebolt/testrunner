package pluginzigbee2mqtt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/slidebolt/sdk-types"
	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestMQTTDiscoveryRoundTrip(t *testing.T) {
	testutil.RequirePlugin(t, "plugin-zigbee2mqtt")

	mqttURL := testutil.PluginEnv("plugin-zigbee2mqtt", "ZIGBEE2MQTT_MQTT_URL", "Z2M_MQTT_BROKER_URL", "MQTT_URL")
	if mqttURL == "" {
		t.Skip("no MQTT URL configured for plugin-zigbee2mqtt")
	}

	client, disconnect := connectMQTTOrSkip(t, mqttURL)
	defer disconnect()

	nonce := fmt.Sprintf("%d", time.Now().UnixNano())
	deviceKey := "it-" + nonce
	uniqueID := "it-light-" + nonce
	topic := fmt.Sprintf("homeassistant/light/%s/%s/config", deviceKey, uniqueID)
	payload := fmt.Sprintf(`{"name":"IT Light %s","unique_id":"%s","device":{"identifiers":["%s"],"name":"IT Device %s","model":"it-model","manufacturer":"it"},"state_topic":"zigbee2mqtt/%s","command_topic":"zigbee2mqtt/%s/set","payload_on":"ON","payload_off":"OFF","value_template":"{{ value_json['state'] }}"}`, nonce, uniqueID, deviceKey, nonce, deviceKey, deviceKey)

	token := client.Publish(topic, 1, true, payload)
	if ok := token.WaitTimeout(3 * time.Second); !ok || token.Error() != nil {
		t.Fatalf("failed publishing MQTT discovery payload: %v", token.Error())
	}

	deviceID := "z2m-device-" + sanitizeForExpectedID(deviceKey)
	waitForDiscoveredDevice(t, deviceKey, deviceID, 10*time.Second)
	waitForDiscoveredEntity(t, deviceID, "z2m-entity-"+sanitizeForExpectedID(uniqueID), 10*time.Second)
}

func waitForDiscoveredDevice(t *testing.T, sourceID, expectedID string, timeout time.Duration) {
	t.Helper()
	client := http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(testutil.APIBaseURL() + "/api/plugins/plugin-zigbee2mqtt/devices")
		if err == nil && resp.StatusCode == http.StatusOK {
			var devices []types.Device
			if decodeErr := json.NewDecoder(resp.Body).Decode(&devices); decodeErr == nil {
				resp.Body.Close()
				for _, d := range devices {
					if d.SourceID == sourceID && d.ID == expectedID {
						return
					}
				}
			} else {
				resp.Body.Close()
			}
		} else if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("discovered device not visible via API: source_id=%q id=%q", sourceID, expectedID)
}

func waitForDiscoveredEntity(t *testing.T, deviceID, entityID string, timeout time.Duration) {
	t.Helper()
	client := http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	url := testutil.APIBaseURL() + "/api/plugins/plugin-zigbee2mqtt/devices/" + deviceID + "/entities"
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			var entities []types.Entity
			if decodeErr := json.NewDecoder(resp.Body).Decode(&entities); decodeErr == nil {
				resp.Body.Close()
				for _, e := range entities {
					if e.ID == entityID {
						return
					}
				}
			} else {
				resp.Body.Close()
			}
		} else if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("discovered entity not visible via API: device_id=%q entity_id=%q", deviceID, entityID)
}

func connectMQTTOrSkip(t *testing.T, url string) (mqtt.Client, func()) {
	t.Helper()
	opts := mqtt.NewClientOptions().AddBroker(url)
	opts.SetClientID(fmt.Sprintf("z2m-integration-test-%d", time.Now().UnixNano()))
	opts.SetAutoReconnect(false)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if ok := token.WaitTimeout(4 * time.Second); !ok || token.Error() != nil {
		t.Skipf("MQTT broker unavailable at %q: %v", url, token.Error())
	}
	return client, func() { client.Disconnect(250) }
}

func sanitizeForExpectedID(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "unknown"
	}
	var b strings.Builder
	lastDash := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		isAZ09 := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if isAZ09 {
			b.WriteByte(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}
