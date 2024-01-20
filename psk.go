package rgap

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type PSK [32]byte

func (psk *PSK) AsSlice() []byte {
	return psk[:]
}

func (psk *PSK) AsHexString() string {
	return hex.EncodeToString(psk.AsSlice())
}

func (psk *PSK) String() string {
	return psk.AsHexString()
}

func GeneratePSK() (PSK, error) {
	var psk PSK
	if _, err := rand.Read(psk.AsSlice()); err != nil {
		return psk, fmt.Errorf("unable to generate random bytes for PSK: %w", err)
	}
	return psk, nil
}
