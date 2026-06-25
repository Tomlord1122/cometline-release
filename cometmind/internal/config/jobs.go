package config

// JobNotificationSettings controls job status notifications.
type JobNotificationSettings struct {
	Enabled     bool `json:"enabled" mapstructure:"enabled"`
	OnClaimed   bool `json:"on_claimed" mapstructure:"on_claimed"`
	OnCompleted bool `json:"on_completed" mapstructure:"on_completed"`
	OnReleased  bool `json:"on_released" mapstructure:"on_released"`
}

// JobSettings holds runtime job configuration.
type JobSettings struct {
	Notifications      JobNotificationSettings `json:"notifications"`
	LeaseMinutes       int                     `json:"lease_minutes"`
	DeletedPurgeDays   int                     `json:"deleted_purge_days"`
	ReconcileIntervalS int                     `json:"reconcile_interval_seconds"`
}

// DefaultJobSettings returns the default job settings.
func DefaultJobSettings() JobSettings {
	return JobSettings{
		Notifications: JobNotificationSettings{
			Enabled:     true,
			OnClaimed:   true,
			OnCompleted: true,
			OnReleased:  false,
		},
		LeaseMinutes:       30,
		DeletedPurgeDays:   30,
		ReconcileIntervalS: 120,
	}
}

// JobsConfig controls the global jobs queue.
type JobsConfig struct {
	Notifications            JobNotificationSettings `json:"notifications" mapstructure:"notifications"`
	LeaseMinutes             int                     `json:"lease_minutes" mapstructure:"lease_minutes"`
	DeletedPurgeDays         int                     `json:"deleted_purge_days" mapstructure:"deleted_purge_days"`
	ReconcileIntervalSeconds int                     `json:"reconcile_interval_seconds" mapstructure:"reconcile_interval_seconds"`
}

func defaultJobsConfig() JobsConfig {
	s := DefaultJobSettings()
	return JobsConfig{
		Notifications:            s.Notifications,
		LeaseMinutes:             s.LeaseMinutes,
		DeletedPurgeDays:         s.DeletedPurgeDays,
		ReconcileIntervalSeconds: s.ReconcileIntervalS,
	}
}

// JobsSettings returns runtime job settings with defaults applied.
func (c *Config) JobsSettings() JobSettings {
	if c == nil {
		return DefaultJobSettings()
	}
	def := DefaultJobSettings()
	j := c.Jobs
	s := JobSettings{
		Notifications:      j.Notifications,
		LeaseMinutes:       j.LeaseMinutes,
		DeletedPurgeDays:   j.DeletedPurgeDays,
		ReconcileIntervalS: j.ReconcileIntervalSeconds,
	}
	if s.LeaseMinutes <= 0 {
		s.LeaseMinutes = def.LeaseMinutes
	}
	if s.DeletedPurgeDays <= 0 && j.DeletedPurgeDays != 0 {
		s.DeletedPurgeDays = def.DeletedPurgeDays
	}
	if s.DeletedPurgeDays == 0 && c.Storage.DeletedJobPurgeDays > 0 {
		s.DeletedPurgeDays = c.Storage.DeletedJobPurgeDays
	}
	if s.DeletedPurgeDays <= 0 {
		s.DeletedPurgeDays = def.DeletedPurgeDays
	}
	if s.ReconcileIntervalS <= 0 {
		s.ReconcileIntervalS = def.ReconcileIntervalS
	}
	if s.Notifications == (JobNotificationSettings{}) {
		s.Notifications = def.Notifications
	}
	return s
}
