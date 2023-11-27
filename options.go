package tailreader

import "time"

type Options struct {
	// WaitForFile indicates whether the reader should wait for the file to be created
	// If this is set to false, Read will return an error if the file does not exist.
	//
	// This will also cause the reader to wait if the file is deleted at some point
	// and CloseOnDelete is set to false.
	WaitForFile bool

	// WaitForFileTimeout indicates how long the reader should wait for the file to be created
	// If this is set to 0, the reader will wait indefinitely.
	WaitForFileTimeout time.Duration

	// CloseOnDelete indicates whether the reader should be closed if the file is deleted
	CloseOnDelete bool

	// CloseOnTruncate indicates whether the reader should be closed if the file is truncated
	CloseOnTruncate bool

	// IdleTimeout indicates how long the reader should wait for new data before closing
	// If this is set to 0, the reader will wait indefinitely
	IdleTimeout time.Duration

	// Whether or not .Read() should return io.EOF if the wait for file or idle timeout is reached
	TreatTimeoutsAsEOF bool
}

type Option func(opts *Options)

func WithWaitForFile(wait bool, timeout time.Duration) Option {
	return func(opts *Options) {
		opts.WaitForFile = wait
		opts.WaitForFileTimeout = timeout
	}
}

func WithCloseOnDelete(close bool) Option {
	return func(opts *Options) {
		opts.CloseOnDelete = close
	}
}

func WithCloseOnTruncate(close bool) Option {
	return func(opts *Options) {
		opts.CloseOnTruncate = close
	}
}

func WithIdleTimeout(timeout time.Duration) Option {
	return func(opts *Options) {
		opts.IdleTimeout = timeout
	}
}

func WithTimeoutsAsEOF(timeoutsAsEOF bool) Option {
	return func(opts *Options) {
		opts.TreatTimeoutsAsEOF = timeoutsAsEOF
	}
}
