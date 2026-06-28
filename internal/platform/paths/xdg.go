// Package paths resolves filesystem locations used by mnemonic.
//
// XDG base directories are resolved through github.com/adrg/xdg, which
// implements the XDG Base Directory Specification with native fallbacks for
// Linux, macOS and Windows. Application-specific MNEMONIC_* overrides and the
// higher-level effective path resolution are layered on top in mnemonic_env.go
// and effective.go.
package paths

import "github.com/adrg/xdg"

// XDGPaths holds the resolved standard XDG base directories.
type XDGPaths struct {
	ConfigHome string
	DataHome   string
	StateHome  string
	CacheHome  string
}

// GetXDGPaths returns the XDG base directories according to the XDG Base
// Directory Specification. Resolution and platform-specific fallbacks are
// delegated to github.com/adrg/xdg. Relative XDG_* values are ignored by the
// specification and therefore fall back to the platform defaults.
//
// Reload is invoked so that environment changes performed after process start
// (e.g. in tests or when the caller reconfigures XDG_* at runtime) are
// reflected without requiring a separate explicit refresh call.
func GetXDGPaths() XDGPaths {
	xdg.Reload()

	return XDGPaths{
		ConfigHome: xdg.ConfigHome,
		DataHome:   xdg.DataHome,
		StateHome:  xdg.StateHome,
		CacheHome:  xdg.CacheHome,
	}
}
