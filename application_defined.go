// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package rtcp

import (
	"encoding/binary"
	"fmt"
)

// ApplicationDefined represents an RTCP application-defined packet.
type ApplicationDefined struct {
	SubType    uint8
	SenderSSRC uint32
	MediaSSRC  uint32
	Name       string
	Data       []byte
}

// DestinationSSRC returns the SSRC value for this packet.
func (a ApplicationDefined) DestinationSSRC() []uint32 {
	return []uint32{a.MediaSSRC}
}

// Marshal serializes the application-defined struct into a byte slice with padding.
func (a ApplicationDefined) Marshal() ([]byte, error) {
	dataLength := len(a.Data)
	if dataLength > 0xFFFF-16 {
		return nil, errAppDefinedDataTooLarge
	}
	if len(a.Name) != 4 {
		return nil, errAppDefinedInvalidName
	}
	// Calculate the padding size to be added to make the packet length a multiple of 4 bytes.
	paddingSize := 4 - (dataLength % 4)
	if paddingSize == 4 {
		paddingSize = 0
	}

	packetSize := a.MarshalSize()
	header := Header{
		Type:    TypeApplicationDefined,
		Length:  uint16((packetSize / 4) - 1),
		Padding: paddingSize != 0,
		Count:   a.SubType,
	}

	headerBytes, err := header.Marshal()
	if err != nil {
		return nil, err
	}

	rawPacket := make([]byte, packetSize)
	copy(rawPacket, headerBytes)
	binary.BigEndian.PutUint32(rawPacket[4:8], a.SenderSSRC)
	copy(rawPacket[8:12], a.Name)
	binary.BigEndian.PutUint32(rawPacket[12:16], a.MediaSSRC)
	copy(rawPacket[16:], a.Data)

	// Add padding if necessary.
	if paddingSize > 0 {
		for i := 0; i < paddingSize; i++ {
			rawPacket[16+dataLength+i] = byte(paddingSize)
		}
	}

	return rawPacket, nil
}

// Unmarshal parses the given raw packet into an application-defined struct, handling padding.
func (a *ApplicationDefined) Unmarshal(rawPacket []byte) error {
	/*
	    0                   1                   2                   3
	    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	   |V=2|P| subtype |   PT=APP=204  |             length            |
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	   |                           SSRC/CSRC                           |
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	   |                          name (ASCII)                         |
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	   |                           MediaSSRC                           |
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	   |                   application-dependent data                ...
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	*/
	header := Header{}
	err := header.Unmarshal(rawPacket)
	if err != nil {
		return err
	}
	if len(rawPacket) < 16 {
		return errPacketTooShort
	}

	if int(header.Length+1)*4 != len(rawPacket) {
		return errAppDefinedInvalidLength
	}

	a.SubType = header.Count
	a.SenderSSRC = binary.BigEndian.Uint32(rawPacket[4:8])
	a.Name = string(rawPacket[8:12])
	a.MediaSSRC = binary.BigEndian.Uint32(rawPacket[12:16])

	// Check for padding.
	paddingSize := 0
	if header.Padding {
		paddingSize = int(rawPacket[len(rawPacket)-1])
		if paddingSize > len(rawPacket)-16 {
			return errWrongPadding
		}
	}

	a.Data = rawPacket[16 : len(rawPacket)-paddingSize]

	return nil
}

// MarshalSize returns the size of the packet once marshaled
func (a *ApplicationDefined) MarshalSize() int {
	dataLength := len(a.Data)
	// Calculate the padding size to be added to make the packet length a multiple of 4 bytes.
	paddingSize := 4 - (dataLength % 4)
	if paddingSize == 4 {
		paddingSize = 0
	}
	return 16 + dataLength + paddingSize
}

func (a ApplicationDefined) String() string {
	out := fmt.Sprintf("ApplicationDefined from %x\n", a.SenderSSRC)
	out += fmt.Sprintf("Subtype: %d, Name: %s, MediaSSRC:%x, Data:0x%X", a.SubType, a.Name, a.MediaSSRC, a.Data)
	return out
}