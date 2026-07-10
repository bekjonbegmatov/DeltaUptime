package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

type SecretsManager struct {
	key []byte
}

func NewSecretsManager(rawKey string) (*SecretsManager, error) {
	if strings.TrimSpace(rawKey) == "" {
		return nil, fmt.Errorf("%w: secrets master key is required", ErrInvalidInput)
	}

	key, err := normalizeMasterKey(rawKey)
	if err != nil {
		return nil, err
	}

	return &SecretsManager{key: key}, nil
}

func (m *SecretsManager) Encrypt(plaintext []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return nil, nil, fmt.Errorf("new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}

	return gcm.Seal(nil, nonce, plaintext, nil), nonce, nil
}

func (m *SecretsManager) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("%w: invalid nonce size", ErrInvalidInput)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: %w", err)
	}
	return plaintext, nil
}

func (m *SecretsManager) Hash(value string) string {
	mac := hmac.New(sha256.New, m.key)
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func normalizeMasterKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}

	sum := sha256.Sum256([]byte(raw))
	key := make([]byte, len(sum))
	copy(key, sum[:])
	return key, nil
}
