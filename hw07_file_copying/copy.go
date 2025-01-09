package main

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrUnsupportedFile       = errors.New("unsupported file")
	ErrOffsetExceedsFileSize = errors.New("offset exceeds file size")
	ErrLimitLessThenZero     = errors.New("limit must be >= 0")
	ErrCantCreateOutputFile  = errors.New("cannot create")
	ErrReadFile              = errors.New("cant read -from file")
	ErrWriteFile             = errors.New("cant write -to file")
)

func ByteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

func displayProgress(total int64, bytesWritten *atomic.Int64, wg *sync.WaitGroup) {
	// Display progress bar
	var percent, i int

	spinner := []string{"⣷", "⣯", "⣟", "⡿", "⢿", "⣻", "⣽", "⣾"}
	colorGreen := "\033[0;32m"
	colorNone := "\033[0m"
	for {
		time.Sleep(time.Millisecond * 100)
		percent = int((float64(bytesWritten.Load()) / float64(total)) * 100)
		fmt.Printf("\r%s%s%s%4d%% complete", colorGreen, spinner[i], colorNone, percent)
		i++
		i = i % len(spinner)
		if bytesWritten.Load() >= total {
			fmt.Println()
			wg.Done()
			break
		}

	}
}

func Copy(fromPath, toPath string, offset, limit int64) error {
	var totalBytes int64
	var bytesWritten atomic.Int64
	wg := sync.WaitGroup{}
	wg.Add(1)
	// Checking limit
	if limit < 0 {
		return ErrLimitLessThenZero
	}
	if limit == 0 {
		limit = math.MaxInt64
	}

	// Checking inFile file
	inFile, err := os.Open(fromPath)
	if err != nil {
		return ErrUnsupportedFile
	}
	sourceStats, _ := inFile.Stat()
	sourceSize := sourceStats.Size()
	defer inFile.Close()

	// Checking offset
	if offset < 0 || offset > sourceSize {
		return ErrOffsetExceedsFileSize
	}

	// Calculating amount to process
	if sourceSize == 0 || sourceSize-offset > limit {
		totalBytes = limit
	} else {
		totalBytes = sourceSize - offset
	}

	// Create out file
	outFile, err := os.Create(toPath)
	if err != nil {
		return ErrCantCreateOutputFile
	}
	defer outFile.Close()

	// Show info and display progress bar
	fmt.Printf("  From: %s\n    To: %s\nOffset: %9s\n Limit: %9s\n Total: %9s\n\n", from, to, ByteCountIEC(offset), ByteCountIEC(limit), ByteCountIEC(totalBytes))
	go displayProgress(totalBytes, &bytesWritten, &wg)

	// Start data copy
	var writeLimit int64
	buf := make([]byte, 512) // Set max buffer size to 512 bytes, same as default bs size in dd
	inFile.Seek(offset, 0)
	for {
		r, rErr := inFile.Read(buf)
		if rErr != nil && rErr != io.EOF {
			return ErrReadFile
		}
		if int64(r) > (limit - bytesWritten.Load()) {
			writeLimit = limit - bytesWritten.Load()
		} else {
			writeLimit = int64(r)
		}

		w, wErr := outFile.Write(buf[:writeLimit])
		if wErr != nil {
			return ErrWriteFile
		}
		bytesWritten.Add(int64(w))

		if rErr == io.EOF || bytesWritten.Load() >= limit {
			wg.Wait()
			return nil
		}
	}
}
