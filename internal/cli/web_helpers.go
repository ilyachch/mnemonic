package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ilyachch/mnemonic/internal/webauth"
)

func openWebAuthStore(explicitPath string) (*webauth.Store, error) {
	authPath, err := webauth.ResolveDBPath(explicitPath)
	if err != nil {
		return nil, err
	}
	return webauth.NewStore(authPath)
}

func resolveWebServeProjects(flagValue string) []string {
	slugs := parseWebProjects(flagValue)
	if len(slugs) > 0 {
		return slugs
	}
	return parseWebProjects(os.Getenv("MNEMONIC_SERVE_PROJECTS"))
}

func parseWebProjects(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	slugs := make([]string, 0, len(parts))
	for _, part := range parts {
		slug := strings.TrimSpace(part)
		if slug != "" {
			slugs = append(slugs, slug)
		}
	}
	return slugs
}

func normalizeWebListenAddr(portFlag, addrFlag string) string {
	if strings.TrimSpace(portFlag) != "" {
		port := strings.TrimSpace(strings.TrimPrefix(portFlag, ":"))
		if port == "" {
			return ""
		}
		return fmt.Sprintf(":%s", port)
	}
	if strings.TrimSpace(addrFlag) != "" {
		return strings.TrimSpace(addrFlag)
	}
	return ""
}
