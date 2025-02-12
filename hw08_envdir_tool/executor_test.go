package main

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func recoverStd(oldStd []*os.File, currStd []**os.File) {
	for i := range len(currStd) {
		*currStd[i] = oldStd[i]
	}
}

func closeFiles(stdFiles []*os.File) {
	for i := range len(stdFiles) {
		err := stdFiles[i].Close()
		checkErr(err, "")
	}
}

func TestRunCmd(t *testing.T) {
	wd := filepath.Join(os.TempDir(), "envExecutor")
	os.Mkdir(wd, 0o755)
	defer cleanWd(wd)
	stdNames := []string{"in", "out", "err"}
	currStd := []**os.File{&os.Stdin, &os.Stdout, &os.Stderr}
	oldStd := []*os.File{os.Stdin, os.Stdout, os.Stderr}
	stdFiles := make([]*os.File, 3)
	t.Run("check std in-out-err & exit code", func(t *testing.T) {
		defer closeFiles(stdFiles)

		toStdIn := "test-in-out"
		toStdErr := "test-err"
		scriptExitCode := "100"

		// creating test script
		scriptFile := filepath.Join(wd, "test.sh")
		s, _ := os.OpenFile(scriptFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
		s.WriteString("#!/usr/bin/env bash\necho $(</dev/stdin);echo \"" + toStdErr + "\" 1>&2;exit " + scriptExitCode)
		s.Close()

		// creating files for std in/out/err then remap into these files
		for i := range len(stdNames) {
			fp := filepath.Join(wd, stdNames[i])
			f, err := os.Create(fp)
			checkErr(err, "")
			*currStd[i] = f
			stdFiles[i] = f
		}

		// write some data to std-in
		_, err := stdFiles[0].WriteString(toStdIn)
		checkErr(err, "")

		// executing script
		cmd := []string{"/bin/bash", scriptFile}
		if scriptExitCode != "0" {
			require.PanicsWithError(t, "exit status "+scriptExitCode, func() { RunCmd(cmd, Environment{}) })
		} else {
			RunCmd(cmd, Environment{})
		}
		recoverStd(oldStd, currStd)

		// read contents of std-out & std-err files then compare with test data
		checksMapping := []string{"", toStdIn, toStdErr}
		for i := 1; i < len(stdFiles); i++ {
			file := stdFiles[i]
			file.Seek(0, 0)
			buf := bufio.NewReader(file)
			bytes, _ := buf.ReadBytes(byte('\n'))
			// filter out delimiter rune from string
			pos := len(bytes) - 1
			require.Equal(t, checksMapping[i], string(bytes[:pos]))
		}
	})
}
