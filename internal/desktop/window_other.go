//go:build !windows && !darwin

package desktop

import "errors"

func Run(*Controller, bool) error {
	return errors.New("La aplicación requiere Windows o macOS.")
}

func RequestClose() {}
