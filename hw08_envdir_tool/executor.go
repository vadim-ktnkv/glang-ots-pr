package main

import (
	"os"
	"os/exec"
)

// RunCmd runs a command + arguments (cmd) with environment variables from env.
func RunCmd(cmd []string, env Environment) (returnCode int) {
	execCMD := exec.Command(cmd[0], cmd[1:]...) //nolint:gosec
	execCMD.Stderr = os.Stderr
	execCMD.Stdin = os.Stdin
	execCMD.Stdout = os.Stdout
	for n, e := range env {
		if e.NeedRemove {
			os.Unsetenv(n)
			continue
		}
		os.Setenv(n, e.Value)
	}
	err := execCMD.Run()
	if err != nil {
		panic(err)
	}
	return execCMD.ProcessState.ExitCode()
}
