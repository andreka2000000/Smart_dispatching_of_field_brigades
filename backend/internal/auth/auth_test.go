package auth

import "testing"

func TestPasswordHashesAreSaltedAndVerified(t *testing.T) {
	first, e := Hash("Test-only-password-2026")
	if e != nil {
		t.Fatal(e)
	}
	second, e := Hash("Test-only-password-2026")
	if e != nil {
		t.Fatal(e)
	}
	if first == second || !Verify(first, "Test-only-password-2026") || Verify(first, "wrong-password") {
		t.Fatal("password derivation failed")
	}
	for _, invalid := range []string{"", "pbkdf2-sha256$1$a$b", "pbkdf2-sha256$600000$bad$bad"} {
		if Verify(invalid, "password") {
			t.Fatal("invalid hash accepted")
		}
	}
}
