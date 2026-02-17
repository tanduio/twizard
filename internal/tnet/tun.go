package tnet

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
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

func OpenRawInterface(device string) (*os.File, error) {
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

type Tun struct {
	tf   *os.File
	list map[int]*watchitem
	mu   sync.Mutex
}

type watchitem struct {
	ctx context.Context
	ch  chan<- []byte
}

func OpenTunInterface(device string) (*Tun, error) {
	if len(device) == 0 {
		return nil, emptyDeviceErr
	}

	tunFile, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		log.Fatal("Failed to open /dev/net/tun:", err)
	}

	fd := tunFile.Fd()

	var req struct {
		Name  [IFNAMSIZ]byte
		Flags uint16
		pad   [0x28 - IFNAMSIZ - 2]byte
	}
	copy(req.Name[:], device)
	req.Flags = IFF_TUN | IFF_NO_PI

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, TUNSETIFF, uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		log.Fatal("ioctl TUNSETIFF failed:", errno)
	}

	return &Tun{
		tf:   tunFile,
		list: make(map[int]*watchitem),
	}, nil
}

func (t *Tun) Close() error {
	return t.tf.Close()
}

func (t *Tun) Send(ctx context.Context, packet []byte) (<-chan []byte, error) {
	respCh := make(chan []byte)

	witem := &watchitem{
		ctx: ctx,
		ch:  respCh,
	}

	ip := NewIPPacket(packet)

	var port int
	if ip.TCP != nil {
		port = int(ip.TCP.SourcePort)
	} else {
		return nil, errors.New("failed to get the source port")
	}

	t.mu.Lock()
	if _, ok := t.list[port]; ok {
		t.mu.Unlock()
		return nil, errors.New("duplicate port in watch list")
	}
	t.list[port] = witem
	t.mu.Unlock()

	_, err := t.tf.Write(packet)
	if err != nil {
		log.Printf("Error writing to TUN: %v", err)
		return nil, err
	}

	return respCh, nil
}

func (t *Tun) StartReader() error {
	buffer := make([]byte, 4000)

	for {
		n, err := t.tf.Read(buffer)
		if err != nil {
			log.Printf("Error reading packet: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if n > 0 {
			packet := buffer[:n]

			ip := NewIPPacket(packet)

			var port int
			if ip.TCP != nil {
				port = int(ip.TCP.DestinationPort)
			} else {
				log.Println("failed to get the destination port")
			}

			t.mu.Lock()
			if witem, ok := t.list[port]; ok {
				resp := make([]byte, n)
				copy(resp, packet)
				witem.ch <- resp

				delete(t.list, port)
			} else {
				t.mu.Unlock()
				log.Println("failed to find watchitem to send response")
				continue
			}
			t.mu.Unlock()
		}
	}
}
