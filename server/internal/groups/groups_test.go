package groups

import (
	"testing"
)

func TestGroupModel(t *testing.T) {
	g := Group{
		ID:        []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Name:      "Test Group",
		CreatorID: []byte{0xaa, 0xbb},
		Settings:  []byte("{}"),
	}
	if g.Name != "Test Group" {
		t.Fatal("name mismatch")
	}
	if len(g.ID) != 16 {
		t.Fatal("ID should be 16 bytes")
	}
}

func TestMemberRoles(t *testing.T) {
	if RoleAdmin != "admin" {
		t.Fatal("admin")
	}
	if RoleModerator != "moderator" {
		t.Fatal("moderator")
	}
	if RoleMember != "member" {
		t.Fatal("member")
	}
}

func TestGroupMemberModel(t *testing.T) {
	m := GroupMember{
		GroupID: []byte{1, 2},
		UserID:  []byte{3, 4},
		Role:    RoleAdmin,
	}
	if m.Role != RoleAdmin {
		t.Fatal("role mismatch")
	}
}

func TestMigrateSQL(t *testing.T) {
	if migrateSQL == "" {
		t.Fatal("migrateSQL should not be empty")
	}
}
