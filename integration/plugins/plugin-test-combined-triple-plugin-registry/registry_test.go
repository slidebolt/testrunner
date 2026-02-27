package plugintestcombinedtripleregistry

import (
	"testing"

	"github.com/slidebolt/testrunner/integration/testutil"
)

func TestAllThreePluginsRegistered(t *testing.T) {
	testutil.RequirePlugins(t, "plugin-test-clean", "plugin-test-slow", "plugin-test-flaky")
}
