package config

import "github.com/cometline/cometmind/internal/jobs"

// JobsConfig controls the global jobs queue.
type JobsConfig struct {
	Notifications          jobs.NotificationSettings `json:"notifications" mapstructure:"notifications"`
	LeaseMinutes           int                       `json:"lease_minutes" mapstructure:"lease_minutes"`
	DeletedPurgeDays       int                       `json:"deleted_purge_days" mapstructure:"deleted_purge_days"`
	ReconcileIntervalSeconds int                     `json:"reconcile_interval_seconds" mapstructure:"reconcile_interval_seconds"`
}

func defaultJobsConfig() JobsConfig {
	s := jobs.DefaultSettings()
	return JobsConfig{
		Notifications:            s.Notifications,
		LeaseMinutes:             s.LeaseMinutes,
		DeletedPurgeDays:         s.DeletedPurgeDays,
		ReconcileIntervalSeconds: s.ReconcileIntervalS,
	}
}

// JobsSettings returns runtime job settings with defaults applied.
func (c *Config) JobsSettings() jobs.Settings {
	if c == nil {
		return jobs.DefaultSettings()
	}
	def := jobs.DefaultSettings()
	j := c.Jobs
	s := jobs.Settings{
		Notifications: j.Notifications,
		LeaseMinutes:  j.LeaseMinutes,
		DeletedPurgeDays: j.DeletedPurgeDays,
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
	if s.Notifications == (jobs.NotificationSettings{}) {
		s.Notifications = def.Notifications
	}
	return s
}
