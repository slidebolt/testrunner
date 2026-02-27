package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	runner "github.com/slidebolt/sdk-runner"
)

func pluginHealthURL(id string) string {
	if id == "" {
		return apiBaseURL() + runner.HealthEndpoint
	}
	return apiBaseURL() + runner.HealthEndpoint + "?id=" + id
}

func waitForPlugin(id string, timeout time.Duration) bool {
	client := http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(pluginHealthURL(id))
		if err == nil && resp.StatusCode == http.StatusOK {
			var status map[string]string
			json.NewDecoder(resp.Body).Decode(&status)
			resp.Body.Close()
			if status["status"] == "perfect" {
				return true
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

func TestGatewayHealth(t *testing.T) {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(pluginHealthURL(""))
	if err != nil {
		t.Fatalf("Gateway unreachable: %v", err)
	}
	defer resp.Body.Close()

	var status map[string]string
	json.NewDecoder(resp.Body).Decode(&status)
	if status["status"] != "perfect" {
		t.Errorf("Gateway reported unhealthy: %v", status)
	}
	fmt.Println("PASS: Gateway is perfect.")
}

func TestGatewaySelfRegistered(t *testing.T) {
	if !waitForPlugin("gateway", 5*time.Second) {
		t.Fatal("gateway did not become healthy within timeout")
	}

	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(pluginHealthURL("gateway"))
	if err != nil {
		t.Fatalf("Gateway plugin health unreachable: %v", err)
	}
	defer resp.Body.Close()

	var status map[string]string
	json.NewDecoder(resp.Body).Decode(&status)
	if status["status"] != "perfect" || status["service"] != "gateway" {
		t.Errorf("gateway self-registration reported unexpected status: %v", status)
	}
	fmt.Println("PASS: Gateway self-registered and healthy.")
}

func TestHTTPConnectivity(t *testing.T) {
	resp, err := http.Get(apiBaseURL() + "/api/plugins")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatal("Gateway not responding on /api/plugins")
	}
	fmt.Println("PASS: Basic gateway connectivity verified.")
}
