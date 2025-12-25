package workflow_session

import "errors"

var (
	ErrLockTimeout        = errors.New("lock acquisition timed out")
	ErrLockUnavailable    = errors.New("lock unavailable")
	ErrMappingUnavailable = errors.New("mapping store unavailable")
)
