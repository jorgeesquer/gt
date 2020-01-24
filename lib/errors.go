package lib

import "errors"

var ErrReadOnlyOrUndefined = errors.New("undefined or readonly property")
var ErrInvalidType = errors.New("invalid value type")
var ErrFileNotFound = errors.New("file not found")
var ErrUnauthorized = errors.New("unauthorized")
