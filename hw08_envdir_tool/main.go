package main

import (
	"os"
)

func main() {
	// Parsing envdir content
	envs, err := ReadDir(os.Args[1])
	if err != nil {
		panic(err)
	}
	// Starting command with env vars
	ret := RunCmd(os.Args[2:], envs)
	os.Exit(ret)
}
