package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestGatewayHealth(t *testing.T) {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(testutil.PluginHealthURL(""))
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
	if !testutil.WaitForPlugin("gateway", 5*time.Second) {
		t.Fatal("gateway did not become healthy within timeout")
	}

	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(testutil.PluginHealthURL("gateway"))
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
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(testutil.APIBaseURL() + "/api/plugins")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatal("Gateway not responding on /api/plugins")
	}
	fmt.Println("PASS: Basic gateway connectivity verified.")
}
