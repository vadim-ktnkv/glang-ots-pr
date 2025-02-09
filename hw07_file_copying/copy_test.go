package main

import (
	"errors"
	"os"
	"testing"
)

func TestCopy(t *testing.T) {
	// Place your code here.
	errMsg := "Function must return error: %s; received: %s\n"
	t.Run("invalid parameters", func(t *testing.T) {
		defInput := "testdata/input.txt"
		defOutput := "/tmp/out"

		s := []struct {
			in     string
			out    string
			offset int64
			limit  int64
			err    error
		}{
			{in: defInput, out: defOutput, offset: -745634, limit: 0, err: ErrOffsetExceedsFileSize},
			{in: defInput, out: defOutput, offset: 0, limit: -745634, err: ErrLimitLessThenZero},
			{in: "", out: defOutput, offset: 0, limit: 0, err: ErrUnsupportedFile},
			{in: os.TempDir(), out: defInput, offset: 0, limit: 0, err: ErrUnsupportedFile},
			{in: defInput, out: "", offset: 0, limit: 0, err: ErrCantCreateOutputFile},
			{in: defInput, out: os.TempDir(), offset: 0, limit: 0, err: ErrCantCreateOutputFile},
			{in: defInput, out: defInput, offset: 0, limit: 0, err: ErrSameFile},
		}

		for _, c := range s {
			err := Copy(c.in, c.out, c.offset, c.limit)
			if !errors.Is(err, c.err) {
				t.Errorf(errMsg, c.err, err)
			}
		}
	})
}
