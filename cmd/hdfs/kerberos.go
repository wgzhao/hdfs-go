package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	krb "github.com/jcmturner/gokrb5/v8/client"
	"github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
)

func getKerberosClient() (*krb.Client, error) {
	configPath, err := kerberosConfigPath()
	if err != nil {
		return nil, err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	ccachePath, err := kerberosCCachePath()
	if err != nil {
		return nil, err
	}

	ccache, err := credentials.LoadCCache(ccachePath)
	if err != nil {
		return nil, fmt.Errorf("couldn't load ccache %s: %w", ccachePath, err)
	}

	client, err := krb.NewFromCCache(ccache, cfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// ccachePathFromEnv parses KRB5CCNAME. Only FILE ccaches can be read by
// gokrb5, but a bare path is accepted for convenience.
func ccachePathFromEnv() (string, bool, error) {
	ccachePath := os.Getenv("KRB5CCNAME")
	if ccachePath == "" {
		return "", false, nil
	}

	if strings.HasPrefix(ccachePath, "FILE:") {
		return strings.TrimPrefix(ccachePath, "FILE:"), true, nil
	}

	// A bare path is usable, including drive-letter paths on Windows.
	if filepath.IsAbs(ccachePath) {
		return ccachePath, true, nil
	}

	// Reject non-FILE ccache types (DIR:, KEYRING:, API:, ...).
	if strings.Contains(ccachePath, ":") {
		return "", false, fmt.Errorf("unusable ccache: %s", ccachePath)
	}

	return ccachePath, true, nil
}
