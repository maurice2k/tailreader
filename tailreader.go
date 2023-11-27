// Package tailreader implements an io.Reader that tails a file.
//
// Tailreader is intended to be used for binary files that are being written
// to by another process; it does not implement any kind of "line" reading
// and thus is fully compatible with io.Reader interface and agnostic to the
// file's content.
//
// Copyright 2023 Moritz Fain
// Moritz Fain <moritz@fain.io>
//
// Source available at github.com/maurice2k/tailreader,
// licensed under the MIT license (see LICENSE file).

package tailreader

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

var DefaultOptions = []Option{
	WithWaitForFile(true, 0),
	WithCloseOnDelete(false),
}

type TailingReader struct {
	file     *os.File
	filePath string
	options  *Options
	watcher  *fsnotify.Watcher
	offset   int64
}

var ErrIdleTimeout = fmt.Errorf("idle timeout")
var ErrWaitTimeout = fmt.Errorf("wait for file timeout")
var errTimeout = fmt.Errorf("timeout")

func NewTailingReader(filePath string, options ...Option) (*TailingReader, error) {
	var err error

	tr := &TailingReader{
		filePath: filePath,
		options:  &Options{},
	}

	if len(options) == 0 {
		options = DefaultOptions
	}
	for _, option := range options {
		option(tr.options)
	}

	tr.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	path := filepath.Dir(filePath)
	err = tr.watcher.Add(path)
	if err != nil {
		return nil, err
	}

	return tr, nil
}

func (r *TailingReader) Close() error {
	err := r.watcher.Close()
	r.watcher = nil
	if err != nil {
		return err
	}
	return r.closeFile()
}

func (r *TailingReader) openFile() error {
	if r.file != nil {
		return nil
	}

	file, err := os.Open(r.filePath)
	if err != nil {
		return err
	}

	r.file = file
	r.offset = 0

	return nil
}

func (r *TailingReader) closeFile() error {
	if r.file == nil {
		return nil
	}

	err := r.file.Close()
	r.file = nil
	r.offset = 0

	if err != nil {
		return err
	}

	return nil
}

func (r *TailingReader) getFileSize() (int64, error) {
	fileInfo, err := os.Stat(r.filePath)
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), nil
}

func (r *TailingReader) WaitForFile() error {
	_, err := r.waitForFile(true)
	return err
}

func (r *TailingReader) waitForFile(forceWait bool) (int64, error) {
	for {
		size, err := r.getFileSize()
		if err == nil {
			// file exists, return its size
			return size, nil
		}

		// file does not exist

		if !r.options.WaitForFile && !forceWait {
			// if we don't want to wait for the file, return an error
			return 0, err
		}

		if r.file != nil {
			// the file was already opened, but somehow disappeared

			if r.options.CloseOnDelete {
				_ = r.closeFile()
				return 0, io.EOF
			}
		}

		// wait for the file to be created
		err, _ = r.waitForEventWithTimeout(fsnotify.Create, r.options.WaitForFileTimeout)
		if errors.Is(err, errTimeout) {
			if r.options.TreatTimeoutsAsEOF {
				return 0, io.EOF
			}
			err = ErrWaitTimeout
		}

		if err != nil {
			return 0, err
		}
	}
}

func (r *TailingReader) Read(p []byte) (n int, err error) {
	for {
		size, err := r.waitForFile(false)
		if err != nil {
			return 0, err
		}

		if r.offset > size {
			// file was (most likely) truncated

			_ = r.closeFile()

			if r.options.CloseOnTruncate {
				return 0, io.EOF
			}
		}

		if r.offset < size {
			// we have new data to read

			err = r.openFile()
			if err != nil {
				return 0, err
			}

			n, err = r.file.Read(p)
			if err != nil && err != io.EOF {
				return 0, err
			}

			if n > 0 {
				r.offset += int64(n)
				return n, nil
			}
		}

		// wait for changes to the file (fsnotify.Chmod is triggered on truncate)
		err, event := r.waitForEventWithTimeout(fsnotify.Write|fsnotify.Remove|fsnotify.Rename|fsnotify.Chmod, r.options.IdleTimeout)

		if errors.Is(err, errTimeout) {
			if r.options.TreatTimeoutsAsEOF {
				return 0, io.EOF
			}
			err = ErrIdleTimeout
		}

		if err != nil {
			return 0, err
		}

		if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
			if r.options.CloseOnDelete {
				return 0, io.EOF
			}
			_ = r.closeFile()
		}
	}
}

func (r *TailingReader) waitForEventWithTimeout(eventType fsnotify.Op, timeout time.Duration) (error, fsnotify.Op) {
	var c <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		c = timer.C
	}

	for {
		select {
		case event := <-r.watcher.Events:
			if eventType&event.Op == event.Op && event.Name == r.filePath {
				//fmt.Fprintf(os.Stdout, "event: %v -- file: %s\n", event.Op, event.Name)
				return nil, event.Op
			}
		case err := <-r.watcher.Errors:
			return err, 0
		case <-c:
			return errTimeout, 0
		}
	}
}
