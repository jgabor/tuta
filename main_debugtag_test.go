//go:build debug

package main

import "testing"

// TestDebugCLIWiredUnderDebugTag confirms the production wiring is present
// when tuta is built with -tags debug (mage test, mage build -debug=true, mage
// install). The complementary release-build path — where debugCLI is nil and
// run() explains the requirement — is covered by TestRunDebugUnavailableInReleaseBuild.
func TestDebugCLIWiredUnderDebugTag(t *testing.T) {
	if debugCLI == nil {
		t.Fatal("debugCLI must be wired when built with -tags debug")
	}
}
