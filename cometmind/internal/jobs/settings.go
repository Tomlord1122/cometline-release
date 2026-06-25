package jobs

import "github.com/cometline/cometmind/internal/config"

// NotificationSettings is the legacy name for config.JobNotificationSettings.
// It is kept as a type alias so existing callers continue to compile while the
// canonical type lives in config (breaking the previous config -> jobs import).
type NotificationSettings = config.JobNotificationSettings

// Settings is the legacy name for config.JobSettings.
type Settings = config.JobSettings

// DefaultSettings returns the default job settings.
func DefaultSettings() Settings {
	return config.DefaultJobSettings()
}
