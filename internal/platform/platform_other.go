//go:build !windows && !darwin

package platform

import (
	"fmt"
	"os"
)

func ShowError(message string) error {
	_, err := fmt.Fprintln(os.Stderr, message)
	return err
}
