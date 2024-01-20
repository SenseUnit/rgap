package rgap

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
)

const (
	SignaturePrefix        = "RGAP announce"
	SignatureSize          = 32
	V1              uint16 = 0x0100
)

var SignaturePrefixBytes = []byte(SignaturePrefix)

type AnnouncementData struct {
	Version          uint16
	RedundancyID     uint64
	Timestamp        int64
	AnnouncedAddress [16]byte
}

var announcementDataSize = binary.Size(new(AnnouncementData))

func (ad *AnnouncementData) MarshalBinary() (data []byte, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, announcementDataSize))
	if err := binary.Write(buf, binary.BigEndian, ad); err != nil {
		return nil, fmt.Errorf("binary marshaling of announcement data failed: %w", err)
	}
	return buf.Bytes(), nil
}

func (ad *AnnouncementData) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	if err := binary.Read(buf, binary.BigEndian, ad); err != nil {
		return fmt.Errorf("binary unmarshaling of announcement data failed: %w", err)
	}
	return nil
}

func (ad *AnnouncementData) CalculateSignature(key PSK) ([SignatureSize]byte, error) {
	h := hmac.New(sha256.New, key.AsSlice())
	h.Write([]byte(SignaturePrefixBytes))
	if err := binary.Write(h, binary.BigEndian, ad); err != nil {
		return [SignatureSize]byte{}, fmt.Errorf("announcement data signing failed: %w", err)
	}
	var sig [SignatureSize]byte
	copy(sig[:], h.Sum(nil))
	return sig, nil
}

func (a *AnnouncementData) String() string {
	return fmt.Sprintf("AnnouncementData<Version: %x RedundancyID: %d Timestamp: %d AnnouncedAddress: %s>",
		a.Version, a.RedundancyID, a.Timestamp, net.IP(a.AnnouncedAddress[:]))
}

type Announcement struct {
	Data      AnnouncementData
	Signature [SignatureSize]byte
}

var announcementSize = binary.Size(new(Announcement))

func (a *Announcement) MarshalBinary() (data []byte, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, announcementSize))
	if err := binary.Write(buf, binary.BigEndian, a); err != nil {
		return nil, fmt.Errorf("binary marshaling of announcement failed: %w", err)
	}
	return buf.Bytes(), nil
}

func (a *Announcement) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	if err := binary.Read(buf, binary.BigEndian, a); err != nil {
		return fmt.Errorf("binary unmarshaling of announcement failed: %w", err)
	}
	return nil
}

func (a *Announcement) CheckSignature(key PSK) (bool, error) {
	sig, err := a.Data.CalculateSignature(key)
	if err != nil {
		return false, fmt.Errorf("signature verification failed: %w", err)
	}
	return hmac.Equal(sig[:], a.Signature[:]), nil
}

func (a *Announcement) String() string {
	return fmt.Sprintf("Announcement<Data: %s, Signature: %x>", a.Data.String(), a.Signature)
}
