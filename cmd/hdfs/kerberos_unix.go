//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/user"
)

func kerberosConfigPath() (string, error) {
	if path := os.Getenv("KRB5_CONFIG"); path != "" {
		return path, nil
	}

	return "/etc/krb5.conf", nil
}

func kerberosCCachePath() (string, error) {
	if path, ok, err := ccachePathFromEnv(); ok || err != nil {
		return path, err
	}

	u, err := user.Current()
	if err != nil {
		return "", err
	}

	// This is the default FILE ccache location used by MIT Kerberos. macOS
	// may use an in-memory API cache by default; set KRB5CCNAME to a FILE
	// ccache in that case.
	return fmt.Sprintf("/tmp/krb5cc_%s", u.Uid), nil
}
