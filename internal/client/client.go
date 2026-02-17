package client

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/tanduio/twizard/internal/tnet"
)

func ReadPackets(tunFile *os.File) {
	buffer := make([]byte, 1500)

	for {
		n, err := tunFile.Read(buffer)
		if err != nil {
			log.Printf("Error reading packet: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if n > 0 {
			processPacket(buffer[:n])
		}
	}
}

func processPacket(data []byte) {
	if len(data) < 20 {
		return
	}

	version := data[0] >> 4
	if version != 4 {
		return
	}

	srcIP := fmt.Sprintf("%d.%d.%d.%d", data[12], data[13], data[14], data[15])
	dstIP := fmt.Sprintf("%d.%d.%d.%d", data[16], data[17], data[18], data[19])
	protocol := data[9]

	if protocol == 6 {
		log.Printf("%s -> %s | Proto: %d | Size: %d bytes\ndata: %v\n",
			srcIP, dstIP, protocol, len(data), data)
	}
}

func ForwardTunToTCP(tun *os.File, conn net.Conn) {
	buffer := make([]byte, 1500)

	for {
		n, err := tun.Read(buffer)
		if err != nil {
			log.Printf("Error reading from TUN: %v", err)
			continue
		}

		packet := buffer[:n]

		proto := packet[9]

		if proto != 6 {
			continue
		}

		_, err = conn.Write(packet)
		if err != nil {
			log.Printf("Error sending to server: %v", err)
			return
		}
	}
}

func ForwardTCPToTun(conn net.Conn, tun *os.File) {
	buffer := make([]byte, 4000)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("Error reading from server: %v", err)
			return
		}

		packet := buffer[:n]

		ipPacket := packet

		// Update source IP
		copy(ipPacket[16:20], net.ParseIP("192.168.99.1").To4())

		// Recalculate IP checksum
		ipChecksum := tnet.CalculateIPChecksum(ipPacket[:20])
		binary.BigEndian.PutUint16(ipPacket[10:12], ipChecksum)

		// Recalculate TCP checksum
		srcIP := ipPacket[12:16]
		dstIP := ipPacket[16:20]
		tcpData := ipPacket[20:]
		tcpChecksum := tnet.CalculateTCPChecksum(tcpData, srcIP, dstIP)
		binary.BigEndian.PutUint16(ipPacket[36:38], tcpChecksum)

		if len(packet) >= 20 {
			src := fmt.Sprintf("%d.%d.%d.%d", packet[12], packet[13], packet[14], packet[15])
			dst := fmt.Sprintf("%d.%d.%d.%d", packet[16], packet[17], packet[18], packet[19])
			proto := packet[9]
			log.Printf("[←] TCP->TUN: %s → %s (proto:%d, %d bytes)\n", src, dst, proto, n)
		}

		_, err = tun.Write(ipPacket)
		if err != nil {
			log.Printf("Error writing to TUN: %v", err)
		}
	}
}
