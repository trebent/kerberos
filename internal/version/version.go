package version

import "runtime/debug"

// Exposed to have a settable variable when building binaries.
var Ver = ""

func init() {
	// This is used as a way of supporting EITHER runtime BuildInfo, OR settings the
	// version during build time. If Version is empty, we try to fetch it from BuildInfo,
	// if it's still not found there and unset, "unknown" will be used.
	if Ver == "" {
		info, ok := debug.ReadBuildInfo()

		if ok && info.Main.Version != "" {
			Ver = info.Main.Version
		}
	}

	if Ver == "" {
		Ver = "unknown"
	}
}

// Version returns the semver build version of Kerberos, or "unknown" if no version could be
// detected.
func Version() string {
	return Ver
}
