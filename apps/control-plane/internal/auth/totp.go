package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	totpDigits = 6
	totpPeriod = 30
)

type TOTPManager struct {
	issuer string
}

func NewTOTPManager(issuer string) *TOTPManager {
	if strings.TrimSpace(issuer) == "" {
		issuer = "DeltaUptime"
	}
	return &TOTPManager{issuer: issuer}
}

func (m *TOTPManager) GenerateSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate totp secret: %w", err)
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(secret), "="), nil
}

func (m *TOTPManager) VerifyCode(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != totpDigits {
		return false
	}

	for _, offset := range []int64{-1, 0, 1} {
		counter := (now.Unix() / totpPeriod) + offset
		if counter < 0 {
			continue
		}
		if m.codeAt(secret, counter) == code {
			return true
		}
	}

	return false
}

func (m *TOTPManager) OTPAuthURL(accountName, secret string) string {
	label := url.PathEscape(fmt.Sprintf("%s:%s", m.issuer, accountName))
	values := url.Values{}
	values.Set("secret", secret)
	values.Set("issuer", m.issuer)
	values.Set("algorithm", "SHA1")
	values.Set("digits", strconv.Itoa(totpDigits))
	values.Set("period", strconv.Itoa(totpPeriod))
	return "otpauth://totp/" + label + "?" + values.Encode()
}

func (m *TOTPManager) codeAt(secret string, counter int64) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return ""
	}

	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(counter))

	h := hmac.New(sha1.New, key)
	_, _ = h.Write(buf[:])
	sum := h.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	binCode := binary.BigEndian.Uint32(sum[offset : offset+4])
	binCode &= 0x7fffffff
	return fmt.Sprintf("%06d", int(binCode)%1000000)
}
