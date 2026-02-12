package domain

import (
	"testing"
	"time"
)

func TestNewUser_ValidInput(t *testing.T) {
	u, err := NewUser("auth0|abc123", "Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if u.ID().IsZero() {
		t.Fatal("expected non-zero ID")
	}
	if u.ExternalID() != "auth0|abc123" {
		t.Errorf("external ID = %q, want %q", u.ExternalID(), "auth0|abc123")
	}
	if u.DisplayName() != "Alice" {
		t.Errorf("display name = %q, want %q", u.DisplayName(), "Alice")
	}
	if u.Email() != "alice@example.com" {
		t.Errorf("email = %q, want %q", u.Email(), "alice@example.com")
	}
	if u.AvatarURL() != "" {
		t.Errorf("avatar URL = %q, want empty", u.AvatarURL())
	}
	if u.Timestamps().CreatedAt.IsZero() {
		t.Fatal("expected non-zero created_at")
	}
}

func TestNewUser_TrimsWhitespace(t *testing.T) {
	u, err := NewUser("  auth0|abc123  ", "  Alice  ", "  alice@example.com  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if u.ExternalID() != "auth0|abc123" {
		t.Errorf("external ID = %q, want %q", u.ExternalID(), "auth0|abc123")
	}
	if u.DisplayName() != "Alice" {
		t.Errorf("display name = %q, want %q", u.DisplayName(), "Alice")
	}
	if u.Email() != "alice@example.com" {
		t.Errorf("email = %q, want %q", u.Email(), "alice@example.com")
	}
}

func TestNewUser_RejectsEmptyExternalID(t *testing.T) {
	_, err := NewUser("", "Alice", "alice@example.com")
	if err == nil {
		t.Fatal("expected error for empty external ID")
	}
}

func TestNewUser_RejectsWhitespaceOnlyExternalID(t *testing.T) {
	_, err := NewUser("   ", "Alice", "alice@example.com")
	if err == nil {
		t.Fatal("expected error for whitespace-only external ID")
	}
}

func TestNewUser_RejectsEmptyDisplayName(t *testing.T) {
	_, err := NewUser("auth0|abc123", "", "alice@example.com")
	if err == nil {
		t.Fatal("expected error for empty display name")
	}
}

func TestNewUser_RejectsEmptyEmail(t *testing.T) {
	_, err := NewUser("auth0|abc123", "Alice", "")
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestNewUser_GeneratesUniqueIDs(t *testing.T) {
	u1, _ := NewUser("auth0|1", "Alice", "alice@example.com")
	u2, _ := NewUser("auth0|2", "Bob", "bob@example.com")

	if u1.ID().String() == u2.ID().String() {
		t.Fatal("expected unique IDs for different users")
	}
}

func TestReconstructUser_PreservesAllFields(t *testing.T) {
	original, _ := NewUser("auth0|abc123", "Alice", "alice@example.com")

	reconstructed := ReconstructUser(
		original.ID(),
		original.ExternalID(),
		original.DisplayName(),
		original.Email(),
		"https://example.com/avatar.png",
		original.Timestamps(),
	)

	if reconstructed.ID() != original.ID() {
		t.Errorf("ID = %v, want %v", reconstructed.ID(), original.ID())
	}
	if reconstructed.ExternalID() != original.ExternalID() {
		t.Errorf("external ID = %q, want %q", reconstructed.ExternalID(), original.ExternalID())
	}
	if reconstructed.AvatarURL() != "https://example.com/avatar.png" {
		t.Errorf("avatar URL = %q, want %q", reconstructed.AvatarURL(), "https://example.com/avatar.png")
	}
}

func TestUpdateProfile_ChangesDisplayName(t *testing.T) {
	u, _ := NewUser("auth0|abc123", "Alice", "alice@example.com")

	if err := u.UpdateProfile("Alicia", "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if u.DisplayName() != "Alicia" {
		t.Errorf("display name = %q, want %q", u.DisplayName(), "Alicia")
	}
}

func TestUpdateProfile_ChangesEmail(t *testing.T) {
	u, _ := NewUser("auth0|abc123", "Alice", "alice@example.com")

	if err := u.UpdateProfile("", "newalice@example.com", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if u.Email() != "newalice@example.com" {
		t.Errorf("email = %q, want %q", u.Email(), "newalice@example.com")
	}
}

func TestUpdateProfile_ChangesAvatarURL(t *testing.T) {
	u, _ := NewUser("auth0|abc123", "Alice", "alice@example.com")

	if err := u.UpdateProfile("", "", "https://example.com/new.png"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if u.AvatarURL() != "https://example.com/new.png" {
		t.Errorf("avatar URL = %q, want %q", u.AvatarURL(), "https://example.com/new.png")
	}
}

func TestUpdateProfile_ClearsAvatarURL(t *testing.T) {
	u, _ := NewUser("auth0|abc123", "Alice", "alice@example.com")
	_ = u.UpdateProfile("", "", "https://example.com/avatar.png")

	if err := u.UpdateProfile("", "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if u.AvatarURL() != "" {
		t.Errorf("avatar URL = %q, want empty", u.AvatarURL())
	}
}

func TestUpdateProfile_MultipleFieldsAtOnce(t *testing.T) {
	u, _ := NewUser("auth0|abc123", "Alice", "alice@example.com")

	if err := u.UpdateProfile("Bob", "bob@example.com", "https://example.com/bob.png"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if u.DisplayName() != "Bob" {
		t.Errorf("display name = %q, want %q", u.DisplayName(), "Bob")
	}
	if u.Email() != "bob@example.com" {
		t.Errorf("email = %q, want %q", u.Email(), "bob@example.com")
	}
	if u.AvatarURL() != "https://example.com/bob.png" {
		t.Errorf("avatar URL = %q, want %q", u.AvatarURL(), "https://example.com/bob.png")
	}
}

func TestUpdateProfile_RejectsNoChanges(t *testing.T) {
	u, _ := NewUser("auth0|abc123", "Alice", "alice@example.com")

	err := u.UpdateProfile("", "", "")
	if err == nil {
		t.Fatal("expected error when nothing changes")
	}
}

func TestUpdateProfile_RejectsSameValues(t *testing.T) {
	u, _ := NewUser("auth0|abc123", "Alice", "alice@example.com")

	err := u.UpdateProfile("Alice", "alice@example.com", "")
	if err == nil {
		t.Fatal("expected error when values are identical")
	}
}

func TestUpdateProfile_TouchesUpdatedAt(t *testing.T) {
	u, _ := NewUser("auth0|abc123", "Alice", "alice@example.com")
	before := u.Timestamps().UpdatedAt.Time()

	// Sleep just enough to ensure clock advances.
	time.Sleep(time.Millisecond)

	if err := u.UpdateProfile("Alicia", "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after := u.Timestamps().UpdatedAt.Time()
	if !after.After(before) {
		t.Errorf("updated_at not advanced: before=%v, after=%v", before, after)
	}
}

func TestUpdateProfile_TrimsWhitespace(t *testing.T) {
	u, _ := NewUser("auth0|abc123", "Alice", "alice@example.com")

	if err := u.UpdateProfile("  Bob  ", "  bob@example.com  ", "  https://example.com/bob.png  "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if u.DisplayName() != "Bob" {
		t.Errorf("display name = %q, want %q", u.DisplayName(), "Bob")
	}
	if u.Email() != "bob@example.com" {
		t.Errorf("email = %q, want %q", u.Email(), "bob@example.com")
	}
	if u.AvatarURL() != "https://example.com/bob.png" {
		t.Errorf("avatar URL = %q, want %q", u.AvatarURL(), "https://example.com/bob.png")
	}
}
