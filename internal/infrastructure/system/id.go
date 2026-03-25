package system

import (
	"crypto/rand"
	"encoding/hex"
)

type IDGenerator struct{}

func NewIDGenerator() IDGenerator {
	return IDGenerator{}
}

func (IDGenerator) NewID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}
