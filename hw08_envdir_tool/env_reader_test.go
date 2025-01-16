package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func checkErr(err error, message string) {
	if message != "" {
		os.Stderr.WriteString(message)
	}
	if err != nil {
		panic(err)
	}
}

func writeEnvFile(filePath, data string) {
	var f *os.File
	var err error
	f, err = os.Create(filePath)
	checkErr(err, "")
	_, err = f.WriteString(data)
	checkErr(err, "")
	err = f.Close()
	checkErr(err, "")
}

func cleanWd(wd string) {
	err := os.RemoveAll(wd)
	checkErr(err, "Please remove "+wd+" manually")
}

func TestReadDir(t *testing.T) {
	wd := filepath.Join(os.TempDir(), "envReader")
	defer cleanWd(wd)
	t.Run("various data", func(t *testing.T) {
		os.Mkdir(wd, 0o755)
		nonValid := "MUST NOT EXIST"
		type testData = struct {
			data     string
			expected string
		}
		envData := make(map[string]testData, 5)
		envData["1TEST"] = testData{data: "", expected: nonValid}
		envData["TEST2"] = testData{data: "abcde\n" + string([]byte{0}) + "xyz", expected: "abcde"}
		envData["TEST=3"] = testData{data: "", expected: nonValid}
		envData["TeSt4"] = testData{data: "line1" + string([]byte{0}) + "line2", expected: "line1\nline2"}
		envData["TEST4"] = testData{data: "=4", expected: "=4"}
		envData["ИМЯ5"] = testData{data: "参数1👩🏿‍🚒参数2", expected: "参数1👩🏿‍🚒参数2"}
		var expSize int
		for _, v := range envData {
			if strings.Compare(v.expected, nonValid) == 0 {
				continue
			}
			expSize++
		}
		for k, v := range envData {
			tf := filepath.Join(wd, k)
			writeEnvFile(tf, v.data)
		}
		os.Mkdir(filepath.Join(wd, "TEST6"), 0o755)

		envs, err := ReadDir(wd)
		require.Equal(t, expSize, len(envs))
		require.NoError(t, err)
		for k, v := range envs {
			require.Equal(t, envData[k].expected, v.Value)
		}
		cleanWd(wd)
	})

	t.Run("Empty files", func(t *testing.T) {
		os.Mkdir(wd, 0o755)
		files := []string{"TEST1", "TesT2", "名称3"}
		for _, n := range files {
			tf := filepath.Join(wd, n)
			writeEnvFile(tf, "")
		}
		envs, err := ReadDir(wd)
		require.Equal(t, len(files), len(envs))
		require.NoError(t, err)
		for _, v := range envs {
			require.Equal(t, v.NeedRemove, true)
		}
		cleanWd(wd)
	})

	t.Run("Wrong path", func(t *testing.T) {
		var err error
		var d Environment
		d, err = ReadDir("/dev/random")
		require.Error(t, err)
		require.Nil(t, d)
		d, err = ReadDir("/proc")
		require.Error(t, err)
		require.Nil(t, d)
	})
}
