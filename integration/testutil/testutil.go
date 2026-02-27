package testutil

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	runner "github.com/slidebolt/sdk-runner"
	"github.com/slidebolt/sdk-types"
)

type runtimeConfig struct {
	APIBaseURL string `json:"api_base_url"`
}

var (
	runtimeOnce sync.Once
	runtimeCfg  runtimeConfig
	runtimeErr  error
)

func loadRuntimeConfig() {
	if v := os.Getenv("TEST_API_BASE_URL"); v != "" {
		runtimeCfg.APIBaseURL = v
		return
	}
	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		runtimePath, foundErr := findRuntimeFile()
		if foundErr == nil {
			data, err := os.ReadFile(runtimePath)
			if err == nil {
				if err := json.Unmarshal(data, &runtimeCfg); err != nil {
					runtimeErr = fmt.Errorf("failed decoding %s: %w", runtimePath, err)
					return
				}
				if runtimeCfg.APIBaseURL == "" {
					runtimeErr = fmt.Errorf("%s missing api_base_url", runtimePath)
					return
				}
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	runtimeErr = errors.New("runtime not defined: set TEST_API_BASE_URL, TEST_RUNTIME_PATH, or provide .build/runtime.json")
}

func findRuntimeFile() (string, error) {
	if runtimePath := strings.TrimSpace(os.Getenv("TEST_RUNTIME_PATH")); runtimePath != "" {
		if _, err := os.Stat(runtimePath); err == nil {
			return runtimePath, nil
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	path := wd
	seen := map[string]struct{}{}
	addCandidate := func(c string) (string, bool) {
		if _, exists := seen[c]; exists {
			return "", false
		}
		seen[c] = struct{}{}
		if _, err := os.Stat(c); err == nil {
			return c, true
		}
		return "", false
	}
	for i := 0; i < 8; i++ {
		candidates := []string{
			filepath.Join(path, ".build", "runtime.json"),
			filepath.Join(path, "test", ".build", "runtime.json"),
			filepath.Join(path, "work", "test", ".build", "runtime.json"),
		}
		for _, candidate := range candidates {
			if hit, ok := addCandidate(candidate); ok {
				return hit, nil
			}
		}
		next := filepath.Dir(path)
		if next == path {
			break
		}
		path = next
	}
	return "", os.ErrNotExist
}

func APIBaseURL() string {
	runtimeOnce.Do(loadRuntimeConfig)
	if runtimeErr != nil {
		panic(runtimeErr)
	}
	return runtimeCfg.APIBaseURL
}

func PluginHealthURL(id string) string {
	if id == "" {
		return APIBaseURL() + runner.HealthEndpoint
	}
	return APIBaseURL() + runner.HealthEndpoint + "?id=" + id
}

func WaitForPlugin(id string, timeout time.Duration) bool {
	client := http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(PluginHealthURL(id))
		if err == nil && resp.StatusCode == http.StatusOK {
			var status map[string]string
			_ = json.NewDecoder(resp.Body).Decode(&status)
			resp.Body.Close()
			if status["status"] == "perfect" {
				return true
			}
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

func RegisteredPlugins() (map[string]types.Registration, error) {
	resp, err := http.Get(APIBaseURL() + "/api/plugins")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var registry map[string]types.Registration
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		return nil, err
	}
	return registry, nil
}

func RequirePlugin(t *testing.T, id string) {
	t.Helper()
	if !waitForRegistration(id, 20*time.Second) {
		t.Skipf("plugin %q not registered; skipping plugin-specific tests", id)
	}
	if !WaitForPlugin(id, 20*time.Second) {
		t.Skipf("plugin %q not healthy within timeout; skipping plugin-specific tests", id)
	}
}

func RequirePlugins(t *testing.T, ids ...string) {
	t.Helper()
	missing := waitForRegistrations(ids, 20*time.Second)
	if len(missing) > 0 {
		t.Skipf("missing required plugin(s): %s; skipping combined test", strings.Join(missing, ", "))
	}

	for _, id := range ids {
		if !WaitForPlugin(id, 20*time.Second) {
			t.Skipf("plugin %q not healthy within timeout; skipping combined test", id)
		}
	}
}

func waitForRegistration(id string, timeout time.Duration) bool {
	return len(waitForRegistrations([]string{id}, timeout)) == 0
}

func waitForRegistrations(ids []string, timeout time.Duration) []string {
	deadline := time.Now().Add(timeout)
	for {
		registry, err := RegisteredPlugins()
		if err == nil {
			missing := make([]string, 0)
			for _, id := range ids {
				if _, ok := registry[id]; !ok {
					missing = append(missing, id)
				}
			}
			if len(missing) == 0 || time.Now().After(deadline) {
				return missing
			}
		}
		if time.Now().After(deadline) {
			return append([]string(nil), ids...)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func PluginEnv(pluginID string, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	values := map[string]string{}
	for _, path := range findPluginEnvFiles(pluginID) {
		fileValues, err := parseDotEnvFile(path)
		if err != nil {
			continue
		}
		for k, v := range fileValues {
			if _, exists := values[k]; exists {
				continue
			}
			if strings.TrimSpace(v) == "" {
				continue
			}
			values[k] = strings.TrimSpace(v)
		}
	}
	for _, key := range keys {
		if v := strings.TrimSpace(values[key]); v != "" {
			return v
		}
	}
	return ""
}

func findPluginEnvFiles(pluginID string) []string {
	configRoot := strings.TrimSpace(os.Getenv("TEST_PLUGIN_CONFIG_ROOT"))
	if configRoot == "" {
		configRoot = filepath.Join("config", "plugins")
	}
	roots := []string{
		filepath.Join(configRoot, pluginID),
		filepath.Join("plugins", pluginID),
		filepath.Join("raw", pluginID),
		filepath.Join(pluginID),
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil
	}
	path := wd
	seen := map[string]struct{}{}
	out := make([]string, 0, 8)
	addIfExists := func(p string) {
		if _, exists := seen[p]; exists {
			return
		}
		if _, err := os.Stat(p); err == nil {
			out = append(out, p)
			seen[p] = struct{}{}
		}
	}

	for i := 0; i < 8; i++ {
		for _, root := range roots {
			fullRoot := filepath.Join(path, root)
			addIfExists(filepath.Join(fullRoot, ".env.local"))
			addIfExists(filepath.Join(fullRoot, ".env"))
		}
		if len(out) > 0 {
			return out
		}
		next := filepath.Dir(path)
		if next == path {
			break
		}
		path = next
	}
	return nil
}

func parseDotEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		i := strings.Index(line, "=")
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.Trim(strings.TrimSpace(line[i+1:]), `"'`)
		if key == "" {
			continue
		}
		out[key] = val
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
