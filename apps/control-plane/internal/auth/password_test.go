package auth

import "testing"

func TestPasswordHasherHashAndVerify(t *testing.T) {
	hasher := PasswordHasher{}

	hash, err := hasher.HashPassword("supersecret1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	ok, err := hasher.VerifyPassword(hash, "supersecret1")
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("VerifyPassword returned false for the correct password")
	}

	ok, err = hasher.VerifyPassword(hash, "wrong-password")
	if err != nil {
		t.Fatalf("VerifyPassword(wrong): %v", err)
	}
	if ok {
		t.Fatal("VerifyPassword returned true for a wrong password")
	}
}
