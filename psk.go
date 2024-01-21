package rgap

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"gopkg.in/yaml.v3"
)

const (
	PSKSize = 32
)

type PSK [PSKSize]byte

func (psk *PSK) AsSlice() []byte {
	return psk[:]
}

func (psk *PSK) AsHexString() string {
	return hex.EncodeToString(psk.AsSlice())
}

func (psk *PSK) FromHexString(s string) error {
	b, err := hex.DecodeString(s)
	if err != nil {
		return fmt.Errorf("PSK hex decoding failed: %w", err)
	}
	if len(b) != PSKSize {
		return fmt.Errorf("incorrect PSK length. Expected %d, got %d", PSKSize, len(b))
	}
	copy(psk.AsSlice(), b)
	return nil
}

func (psk *PSK) String() string {
	return psk.AsHexString()
}

func (psk *PSK) UnmarshalYAML(value *yaml.Node) error {
	var hexval string
	if err := value.Decode(&hexval); err != nil {
		return fmt.Errorf("PSK unmarshaler unable to retrieve hex string from given node: %w", err)
	}
	if err := psk.FromHexString(hexval); err != nil {
		return fmt.Errorf("PSK unmarshaller can't set value from hex string: %w", err)
	}
	return nil
}

func (psk *PSK) MarshalYAML() (interface{}, error) {
	return psk.AsHexString(), nil
}

func GeneratePSK() (PSK, error) {
	var psk PSK
	if _, err := rand.Read(psk.AsSlice()); err != nil {
		return psk, fmt.Errorf("unable to generate random bytes for PSK: %w", err)
	}
	return psk, nil
}
