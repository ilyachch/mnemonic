package buildinfo

var version string

// SetVersion stores the build version injected by the main package.
func SetVersion(v string) {
	version = v
}

// Version returns the build version or the local default for untagged builds.
func Version() string {
	if version == "" {
		return "dev"
	}

	return version
}
