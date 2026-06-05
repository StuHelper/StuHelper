package storage

import "errors"

var ErrDriverNotRegistered = errors.New("storage driver not registered")
var ErrInvalidMountConfig = errors.New("invalid storage mount config")
var ErrMountDisabled = errors.New("storage mount disabled")
