package client

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	TUNSETIFF = 0x400454ca
	IFF_TUN   = 0x0001
	IFF_TAP   = 0x0002
	IFF_NO_PI = 0x1000
	IFNAMSIZ  = 16
)

var (
	emptyDeviceErr = errors.New("device can't be empty")
)

type ifReq struct {
	Name  [IFNAMSIZ]byte
	Flags uint16
	pad   [0x28 - IFNAMSIZ - 2]byte
}

func OpenTunInterface(device string) (*os.File, error) {
	if len(device) == 0 {
		return nil, emptyDeviceErr
	}

	tunFile, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		log.Fatal("Failed to open /dev/net/tun:", err)
	}

	fd := tunFile.Fd()

	var req ifReq
	copy(req.Name[:], device)
	req.Flags = IFF_TUN | IFF_NO_PI

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, TUNSETIFF, uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		log.Fatal("ioctl TUNSETIFF failed:", errno)
	}

	return tunFile, nil
}

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

		if len(packet) >= 20 {
			src := fmt.Sprintf("%d.%d.%d.%d", packet[12], packet[13], packet[14], packet[15])
			dst := fmt.Sprintf("%d.%d.%d.%d", packet[16], packet[17], packet[18], packet[19])
			proto := packet[9]

			if proto != 6 {
				continue
			}

			fmt.Printf("[→] TUN->TCP: %s → %s (proto:%d, %d bytes): version[%d], \n%v\n", src, dst, proto, n, packet[0]>>4, packet)
		}

		_, err = conn.Write(packet)
		if err != nil {
			log.Printf("Error sending to server: %v", err)
			return
		}
	}
}

func ForwardTCPToTun(conn net.Conn, tun *os.File) {
	buffer := make([]byte, 1500)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("Error reading from server: %v", err)
			return
		}

		packet := buffer[:n]

		fmt.Printf("client response; receive [%d] bytes\n", n)

		if len(packet) >= 20 {
			src := fmt.Sprintf("%d.%d.%d.%d", packet[12], packet[13], packet[14], packet[15])
			dst := fmt.Sprintf("%d.%d.%d.%d", packet[16], packet[17], packet[18], packet[19])
			proto := packet[9]
			fmt.Printf("[←] TCP->TUN: %s → %s (proto:%d, %d bytes)\n", src, dst, proto, n)
		}

		_, err = tun.Write(packet)
		if err != nil {
			log.Printf("Error writing to TUN: %v", err)
		}
	}
}
