package constant

import "errors"

var (
	ErrParseFailed         = errors.New("parse failed")
	ErrUnsupportedProtocol = errors.New("unsupported protocol")
	ErrFetchTimeout        = errors.New("fetch timeout")
	ErrInvalidURL          = errors.New("invalid URL")
	ErrInvalidData         = errors.New("invalid data")
	ErrEmptySubscription   = errors.New("empty subscription")
)