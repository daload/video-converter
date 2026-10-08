//go:build !windows

package platform

import "os/exec"

func PrepareCommand(command *exec.Cmd) {}
