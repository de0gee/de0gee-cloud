package cloud

import (
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword generates a bcrypt hash of the password using work factor 14.
func HashPassword(password string) (hash string, err error) {
	bHash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return
	}
	hash = base64.StdEncoding.EncodeToString(bHash)
	return
}

// CheckPasswordHash securely compares a bcrypt hashed password with its possible
// plaintext equivalent.  Returns nil on success, or an error on failure.
func CheckPasswordHash(hash, password string) (err error) {
	bHash, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return
	}
	err = bcrypt.CompareHashAndPassword(bHash, []byte(password))
	return
}
