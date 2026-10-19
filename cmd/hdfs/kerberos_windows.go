//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func kerberosConfigPath() (string, error) {
	if path := os.Getenv("KRB5_CONFIG"); path != "" {
		return path, nil
	}

	// MIT Kerberos for Windows looks in these locations by default.
	var candidates []string
	if programData := os.Getenv("ProgramData"); programData != "" {
		candidates = append(candidates,
			filepath.Join(programData, "MIT", "Kerberos5", "krb5.ini"))
	}
	if systemRoot := os.Getenv("SystemRoot"); systemRoot != "" {
		candidates = append(candidates, filepath.Join(systemRoot, "krb5.ini"))
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("couldn't find krb5.ini; set KRB5_CONFIG")
}

func kerberosCCachePath() (string, error) {
	if path, ok, err := ccachePathFromEnv(); ok || err != nil {
		return path, err
	}

	// gokrb5 can only read FILE ccaches. MIT Kerberos for Windows commonly
	// defaults to an in-memory API cache, so users should set KRB5CCNAME to a
	// FILE ccache (for example, FILE:C:\Users\me\krb5cc).
	username := os.Getenv("USERNAME")
	if username == "" {
		username = "user"
	}

	return filepath.Join(os.TempDir(), "krb5cc_"+username), nil
}
