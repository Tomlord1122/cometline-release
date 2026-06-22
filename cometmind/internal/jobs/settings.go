package jobs

// NotificationSettings controls job status notifications.
type NotificationSettings struct {
	Enabled     bool `json:"enabled"`
	OnClaimed   bool `json:"on_claimed"`
	OnCompleted bool `json:"on_completed"`
	OnReleased  bool `json:"on_released"`
}

// Settings holds runtime job configuration.
type Settings struct {
	Notifications      NotificationSettings `json:"notifications"`
	LeaseMinutes       int                  `json:"lease_minutes"`
	DeletedPurgeDays   int                  `json:"deleted_purge_days"`
	ReconcileIntervalS int                  `json:"reconcile_interval_seconds"`
}

func DefaultSettings() Settings {
	return Settings{
		Notifications: NotificationSettings{
			Enabled:     true,
			OnClaimed:   true,
			OnCompleted: true,
			OnReleased:  false,
		},
		LeaseMinutes:       DefaultLeaseMinutes,
		DeletedPurgeDays:   30,
		ReconcileIntervalS: int(DefaultReconcileInterval.Seconds()),
	}
}
