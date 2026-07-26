package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Keyring struct {
	key []byte
}

func NewKeyring(dbPath string) (*Keyring, error) {
	dir := filepath.Dir(dbPath)
	keyFile := filepath.Join(dir, ".gateway.key")

	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, fmt.Errorf("failed to generate key: %w", err)
		}
		if err := os.WriteFile(keyFile, key, 0600); err != nil {
			return nil, fmt.Errorf("failed to write key file: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to stat key file: %w", err)
	}

	key, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key length: %d", len(key))
	}

	return &Keyring{key: key}, nil
}

func (k *Keyring) Encrypt(plaintext string) string {
	if plaintext == "" {
		return ""
	}

	block, err := aes.NewCipher(k.key)
	if err != nil {
		return plaintext
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return plaintext
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return plaintext
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return "enc:" + base64.StdEncoding.EncodeToString(ciphertext)
}

func (k *Keyring) Decrypt(ciphertext string) string {
	if ciphertext == "" {
		return ""
	}

	if !strings.HasPrefix(ciphertext, "enc:") {
		return ciphertext
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, "enc:"))
	if err != nil {
		return ciphertext
	}

	block, err := aes.NewCipher(k.key)
	if err != nil {
		return ciphertext
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ciphertext
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return ciphertext
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return ciphertext
	}

	return string(plaintext)
}

func (k *Keyring) IsEncrypted(value string) bool {
	return strings.HasPrefix(value, "enc:")
}
