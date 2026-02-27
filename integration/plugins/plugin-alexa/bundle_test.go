package pluginalexa

import (
	"testing"

	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestBundleExists(t *testing.T) {
	const pluginID = "plugin-alexa"
	testutil.RequirePlugin(t, pluginID)

	registry, err := testutil.RegisteredPlugins()
	if err != nil {
		t.Fatalf("failed reading plugin registry: %v", err)
	}
	if _, ok := registry[pluginID]; !ok {
		t.Fatalf("plugin %q missing from registry", pluginID)
	}
}
