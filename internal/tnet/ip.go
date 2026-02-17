package tnet

import (
	"encoding/binary"
)

type IPPacket struct {
	Version        byte
	HLEN           byte
	TypeOfService  byte
	TotalLength    uint16
	Identification uint16
	RES            bool
	DF             bool
	MF             bool
	FragmentOffset uint16
	TimeToLive     byte
	Protocol       byte
	HeaderChecksum uint16
	SourceIP       [4]byte
	DestinationIP  [4]byte
	// TODO options

	TCP  *TCPPacket
	Data []byte
}

type TCPPacket struct {
	SourcePort           uint16
	DestinationPort      uint16
	SequenceNumber       uint32
	AcknowledgmentNumber uint32
	HeaderLength         byte

	// flags
	CWR bool
	ECE bool
	URG bool
	ACK bool
	PSH bool
	RST bool
	SYN bool
	FIN bool

	WindowSize    uint16
	TCPChecksum   uint16
	UrgentPointer uint16
	// TODO options
}

func NewIPPacket(packet []byte) IPPacket {
	if len(packet) < 20 {
		return IPPacket{}
	}

	hlen := packet[0] & 0x0F
	actualHeaderSize := int(hlen) * 4

	var data []byte
	if len(packet) > actualHeaderSize {
		data = make([]byte, len(packet[actualHeaderSize:]))
		copy(data, packet[actualHeaderSize:])
	}

	ipPacket := IPPacket{
		Version:        packet[0] >> 4,
		HLEN:           hlen,
		TypeOfService:  packet[1],
		TotalLength:    uint16(packet[2])<<8 | uint16(packet[3]),
		Identification: uint16(packet[4])<<8 | uint16(packet[5]),
		RES:            packet[6]>>7 == 1,
		DF:             (packet[6]>>6)&1 == 1,
		MF:             (packet[6]>>5)&1 == 1,
		FragmentOffset: uint16(packet[6]&31)<<8 | uint16(packet[7]),
		TimeToLive:     packet[8],
		Protocol:       packet[9],
		HeaderChecksum: uint16(packet[10])<<8 | uint16(packet[11]),
		SourceIP:       [4]byte{packet[12], packet[13], packet[14], packet[15]},
		DestinationIP:  [4]byte{packet[16], packet[17], packet[18], packet[19]},
	}

	if ipPacket.Protocol == 6 {
		ipPacket.TCP = NewTCPPacket(data)
	}

	return ipPacket
}

func NewTCPPacket(packet []byte) *TCPPacket {
	return &TCPPacket{
		SourcePort:           binary.BigEndian.Uint16(packet[0:2]),
		DestinationPort:      binary.BigEndian.Uint16(packet[2:4]),
		SequenceNumber:       binary.BigEndian.Uint32(packet[4:8]),
		AcknowledgmentNumber: binary.BigEndian.Uint32(packet[8:12]),
		// TODO the rest
	}
}

func CalculateIPChecksum(header []byte) uint16 {
	var sum uint32
	length := int(header[0]&0x0F) * 4

	header[10] = 0
	header[11] = 0

	for i := 0; i < length; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(header[i:]))
	}

	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return ^uint16(sum)
}

func CalculateTCPChecksum(data []byte, srcIP, dstIP []byte) uint16 {
	var pseudoHeader []byte

	pseudoHeader = append(pseudoHeader, srcIP...)
	pseudoHeader = append(pseudoHeader, dstIP...)
	pseudoHeader = append(pseudoHeader, 0, 6)
	tcpLen := uint16(len(data))
	pseudoHeader = append(pseudoHeader, byte(tcpLen>>8), byte(tcpLen&0xFF))

	var sum uint32

	for i := 0; i < len(pseudoHeader); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(pseudoHeader[i:]))
	}

	checksumOffset := 16
	savedChecksum := binary.BigEndian.Uint16(data[checksumOffset:])
	binary.BigEndian.PutUint16(data[checksumOffset:], 0)

	for i := 0; i < len(data); i += 2 {
		if i+1 < len(data) {
			sum += uint32(binary.BigEndian.Uint16(data[i:]))
		} else {
			sum += uint32(data[i]) << 8
		}
	}

	binary.BigEndian.PutUint16(data[checksumOffset:], savedChecksum)

	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return ^uint16(sum)
}
