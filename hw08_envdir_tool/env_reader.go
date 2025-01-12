package main

import (
	"bufio"
	"os"
	"strings"
	"unicode"
)

type Environment map[string]EnvValue

// EnvValue helps to distinguish between empty files and files with the first empty line.
type EnvValue struct {
	Value      string
	NeedRemove bool
}

// By IEEE Std 1003.1-2024 -
// names shall not contain any bytes that have the encoded value of the character '='
// and do not begin with a digit.
// Other characters, and byte sequences that do not form valid characters, may be permitted by an implementation;
// Function return TRUE if input string is IEEE Std 1003.1-2024 compliant.
func validateName(name string) bool {
	// 1st rune must not be a digit 0-9
	if name[0] > 47 && name[0] < 58 {
		return false
	}
	for i := 0; i < len(name); i++ {
		// must not contain equals sign
		if name[i] == '=' {
			return false
		}
	}
	return true
}

// ReadDir reads a specified directory and returns map of env variables.
// Variables represented as files where filename is name of variable, file first line is a value.
func ReadDir(dir string) (Environment, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fs := os.DirFS(dir)
	envMap := Environment{}

	for _, entry := range dirEntries {
		if entry.IsDir() || !validateName(entry.Name()) {
			continue
		}

		stat, _ := entry.Info()
		if stat.Size() == 0 {
			envMap[entry.Name()] = EnvValue{Value: "", NeedRemove: true}
			continue
		}

		f, err := fs.Open(entry.Name())
		if err != nil {
			return nil, err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		if !scanner.Scan() {
			return nil, scanner.Err()
		}
		pre := strings.ReplaceAll(scanner.Text(), string(rune(0)), "\n")
		value := strings.TrimRightFunc(pre, unicode.IsSpace)

		envMap[entry.Name()] = EnvValue{Value: value, NeedRemove: false}
	}
	return envMap, nil
}
