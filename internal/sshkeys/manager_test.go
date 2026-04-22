package sshkeys

import "testing"

func TestValidateKeyName(t *testing.T) {
	t.Parallel()

	valid := []string{"default", "work_laptop", "prod.key-01", "A1"}
	for _, name := range valid {
		name := name
		t.Run("valid_"+name, func(t *testing.T) {
			t.Parallel()
			if err := validateKeyName(name); err != nil {
				t.Fatalf("expected %q to be valid: %v", name, err)
			}
		})
	}

	invalid := []string{"", "a b", "../id_rsa", "key$", "key/name"}
	for _, name := range invalid {
		name := name
		t.Run("invalid_"+name, func(t *testing.T) {
			t.Parallel()
			if err := validateKeyName(name); err == nil {
				t.Fatalf("expected %q to be invalid", name)
			}
		})
	}
}
