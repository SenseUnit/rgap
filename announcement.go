package rgap

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	signaturePrefix = "RGAP announce"
)

type AnnouncementData struct {
	Version          uint16
	RedundancyID     uint64
	Timestamp        int64
	AnnouncedAddress [16]byte
}

var announcementDataSize = binary.Size((*AnnouncementData)(nil))

func (ad *AnnouncementData) MarshalBinary() (data []byte, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, announcementDataSize))
	if err := binary.Write(buf, binary.BigEndian, ad); err != nil {
		return nil, fmt.Errorf("binary marshaling of announcement data failed: %w", err)
	}
	return buf.Bytes(), nil
}

type Announcement struct {
	AnnouncementData
	Signature [32]byte
}

var announcementSize = binary.Size((*Announcement)(nil))

func (a *Announcement) MarshalBinary() (data []byte, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, announcementSize))
	if err := binary.Write(buf, binary.BigEndian, a); err != nil {
		return nil, fmt.Errorf("binary marshaling of announcement failed: %w", err)
	}
	return buf.Bytes(), nil
}
