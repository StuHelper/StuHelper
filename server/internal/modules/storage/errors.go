package storage

import "errors"

var ErrDriverNotRegistered = errors.New("storage driver not registered")
var ErrMountDisabled = errors.New("storage mount disabled")
