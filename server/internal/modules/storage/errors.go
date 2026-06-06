package storage

import "errors"

var ErrDriverNotRegistered = errors.New("storage driver not registered")
var ErrInvalidMountConfig = errors.New("invalid storage mount config")
var ErrInvalidMountID = errors.New("invalid storage mount id")
var ErrInvalidObjectKey = errors.New("invalid storage object key")
var ErrMountAlreadyExists = errors.New("storage mount already exists")
var ErrDefaultMountBucketDrift = errors.New("default storage mount bucket drift")
var ErrMountDisabled = errors.New("storage mount disabled")
var ErrStoredObjectMissing = errors.New("storage driver returned no object metadata")
var ErrInvalidStoredObject = errors.New("storage driver returned invalid object metadata")
