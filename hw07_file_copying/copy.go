package main

import (
	"errors"
	"fmt"
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
	ErrCantCreateOutputFile  = errors.New("cannot create output file")
	ErrReadFile              = errors.New("cant read -from file")
	ErrWriteFile             = errors.New("cant write -to file")
	ErrSameFile              = errors.New("can't read and write on the same file")
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
	var percent, frame int

	spinner := []string{"⣷", "⣯", "⣟", "⡿", "⢿", "⣻", "⣽", "⣾"}
	colorGreen := "\033[0;32m"
	colorNone := "\033[0m"
	for {
		time.Sleep(time.Millisecond * 100)
		percent = int((float64(bytesWritten.Load()) / float64(total)) * 100)
		fmt.Printf("\r%s%s%s%4d%% complete", colorGreen, spinner[frame], colorNone, percent)
		frame++
		frame %= len(spinner)
		if bytesWritten.Load() >= total {
			fmt.Println()
			wg.Done()
			break
		}
	}
}

func Copy(fromPath, toPath string, offset, limit int64) error {
	var bytesToWrite int64
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

	// Validating in/out files are'nt same
	if fromPath == toPath {
		return ErrSameFile
	}

	// Checking inFile file
	inFile, err := os.Open(fromPath)
	inFileStats, _ := inFile.Stat()
	if err != nil || inFileStats.IsDir() {
		return ErrUnsupportedFile
	}

	sourceSize := inFileStats.Size()
	defer inFile.Close()

	// Checking offset
	if offset < 0 || offset > sourceSize {
		return ErrOffsetExceedsFileSize
	}

	// Calculating amount for copy
	if sourceSize == 0 || limit < sourceSize-offset {
		bytesToWrite = limit
	} else {
		bytesToWrite = sourceSize - offset
	}

	// Create out file
	outFile, err := os.Create(toPath)
	if err != nil {
		return ErrCantCreateOutputFile
	}
	defer outFile.Close()

	// Show info and display progress bar
	fmt.Printf("  From: %s\n    To: %s\nOffset: %9s\n Limit: %9s\n Total: %9s\n\n",
		from, to, ByteCountIEC(offset), ByteCountIEC(limit), ByteCountIEC(bytesToWrite))
	go displayProgress(bytesToWrite, &bytesWritten, &wg)

	// Start data copy
	buf := make([]byte, 512) // Set max buffer size to 512 bytes, same as default bs size in dd
	inFile.Seek(offset, 0)
	process := true
	var count int
	for process {
		if bytesToWrite > int64(len(buf)) {
			count = len(buf)
		} else {
			count = int(bytesToWrite)
			process = false
		}

		_, err := inFile.Read(buf[:count])
		if err != nil {
			return ErrReadFile
		}
		w, err2 := outFile.Write(buf[:count])
		if err2 != nil {
			return ErrWriteFile
		}
		bytesWritten.Add(int64(w))
		bytesToWrite -= int64(w)
	}
	wg.Wait()
	return nil
}
