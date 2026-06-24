// SPDX-License-Identifier: Apache-2.0

package complytime

// ResolveEnvVars is exported for testing.
var ResolveEnvVars = resolveEnvVars

// ResetDeprecationWarnings clears the deduplication state so tests can
// verify warning output without interference from earlier calls.
func ResetDeprecationWarnings() {
	deprecationWarnedMu.Lock()
	deprecationWarned = make(map[string]bool)
	deprecationWarnedMu.Unlock()
}
