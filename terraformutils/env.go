// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"fmt"
	"os"
)

func SetEnv(key, value string) error {
	if err := os.Setenv(key, value); err != nil {
		return fmt.Errorf("failed to set env %s: %w", key, err)
	}
	return nil
}

func UnsetEnv(key string) error {
	if err := os.Unsetenv(key); err != nil {
		return fmt.Errorf("failed to unset env %s: %w", key, err)
	}
	return nil
}
