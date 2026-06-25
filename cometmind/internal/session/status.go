package session

// DelegationStatus is the canonical state of a delegated child session.
type DelegationStatus string

const (
	DelegationPending            DelegationStatus = "pending"
	DelegationRunning            DelegationStatus = "running"
	DelegationCompleted          DelegationStatus = "completed"
	DelegationFailed             DelegationStatus = "failed"
	DelegationCancelled          DelegationStatus = "cancelled"
	DelegationAwaitingUser       DelegationStatus = "awaiting_user"
	DelegationAwaitingPermission DelegationStatus = "awaiting_permission"
)

// String returns the wire representation of the status.
func (s DelegationStatus) String() string { return string(s) }

// IsActive reports whether the delegation is still in flight.
func (s DelegationStatus) IsActive() bool {
	return s == DelegationPending || s == DelegationRunning
}

// IsTerminal reports whether the delegation has finished.
func (s DelegationStatus) IsTerminal() bool {
	switch s {
	case DelegationCompleted, DelegationFailed, DelegationCancelled:
		return true
	}
	return false
}
