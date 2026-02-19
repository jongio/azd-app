// Package notify provides cross-platform OS notification support.
// This is a thin wrapper re-exporting from azd-core.
package notify

import core "github.com/jongio/azd-core/notify"

// Re-export types from azd-core/notify.
type Notification = core.Notification
type Action = core.Action
type Notifier = core.Notifier
type Config = core.Config

// Re-export functions from azd-core/notify.
var (
	DefaultConfig = core.DefaultConfig
	New           = core.New
)

// Re-export error sentinel values from azd-core/notify.
var (
	ErrNotAvailable       = core.ErrNotAvailable
	ErrPermissionDenied   = core.ErrPermissionDenied
	ErrNotificationFailed = core.ErrNotificationFailed
	ErrTimeout            = core.ErrTimeout
)
