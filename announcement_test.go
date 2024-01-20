package rgap

import (
	"testing"
	"time"
)

func noError(err error) {
	if err != nil {
		panic(err)
	}
}

func TestSizes(t *testing.T) {
	if announcementSize != 66 {
		t.Errorf("announcement size seem to be incorrect: %d != 66", announcementSize)
	}
	if announcementDataSize != 34 {
		t.Errorf("announcement size seem to be incorrect: %d != 34", announcementDataSize)
	}
}

func TestMarshalUnmarshal(t *testing.T) {

	key := Must(GeneratePSK())

	msg := Announcement{
		Data: AnnouncementData{
			Version:          0x0100,
			RedundancyID:     12345678901234567890,
			Timestamp:        time.Now().UnixMicro(),
			AnnouncedAddress: [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 127, 0, 0, 1}, // Replace with actual IP address
		},
	}

	msg.Signature = Must(msg.Data.CalculateSignature(key))
	pkt := Must(msg.MarshalBinary())

	// Display the announcement message
	t.Log(msg.String())
	t.Logf("%x", pkt)

	msg1 := Announcement{}
	noError(msg1.UnmarshalBinary(pkt))
	if res := Must(msg1.CheckSignature(key)); !res {
		t.Error("signature verification failed!")
		return
	}
	if msg1 != msg {
		t.Error("message is not equal to original after serialization/deserialization round trip")
	}
}
