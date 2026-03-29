package senderkeys

// RotationEvent describes what triggered a Sender Key rotation.
type RotationEvent int

const (
	RotationMemberRemoved RotationEvent = iota
	RotationMemberAdded
	RotationTimerExpired
)

// NeedsFullRotation returns true if all members must generate new keys.
// This happens when a member is removed (to prevent the removed member
// from decrypting future messages).
func NeedsFullRotation(event RotationEvent) bool {
	return event == RotationMemberRemoved
}

// NeedsDistribution returns true if existing members need to send their
// current distributions to the new member.
func NeedsDistribution(event RotationEvent) bool {
	return event == RotationMemberAdded
}
