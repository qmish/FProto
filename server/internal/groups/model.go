package groups

import "time"

type MemberRole string

const (
	RoleAdmin     MemberRole = "admin"
	RoleModerator MemberRole = "moderator"
	RoleMember    MemberRole = "member"
)

type Group struct {
	ID        []byte // UUID
	Name      string
	CreatorID []byte
	Settings  []byte // JSON
	CreatedAt time.Time
	UpdatedAt time.Time
}

type GroupMember struct {
	GroupID   []byte
	UserID    []byte
	Role      MemberRole
	JoinedAt  time.Time
}
