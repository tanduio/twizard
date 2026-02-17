# twizard

A lightweight TUN-based forwarding proxy designed mainly for educational purposes. This project explores the fundamentals of VPN-like packet forwarding through a simple, transparent proxy implementation.

> **⚠️  Experimental Status**  
> This is primarily a learning project focused on understanding low-level networking concepts. While functional for basic forwarding, it's not recommended for production use. Expect bugs and missing features.

## Overview

**twizard** implements a simple TUN interface proxy that forwards network traffic between client and server. The current implementation provides basic TCP forwarding as a foundation for exploring more advanced VPN concepts.

## Current Status & Roadmap

### ✅ Implemented
- Basic TUN interface setup
- Simple TCP packet forwarding
- Server-side packet reflection
- Complete direct flow with proper server-side processing

### 🔄 Planned Features
- [ ] UDP forwarding (to avoid TCP meltdown issues)
- [ ] Encryption/decryption layer
- [ ] Connection persistence and management

## Motivation

This project was created to:
- Learn how VPNs work at the packet level
- Understand TUN/TAP interfaces
- Experiment with low-level network programming
- Build a foundation for more advanced networking projects

## Current Limitations

As a learning-focused project, the current implementation has several known limitations:
- Only basic TCP forwarding
- Minimal error handling
- No encryption (traffic is sent in plaintext)
- Not optimized for performance

## Example of usage

### Server-side configuration

Routing configuration:

```bash
sudo ip tuntap add dev tun0 mode tun
sudo ip addr add dev tun0 local 192.168.69.0 remote 192.168.69.1
sudo ip link set dev tun0 up
sudo iptables -t filter -I FORWARD -i tun0 -o eth0 -j ACCEPT
sudo iptables -t filter -I FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT
sudo iptables -t nat -I POSTROUTING -o eth0 -j MASQUERADE
```

Start the server:

```bash
sudo ./twizsrv --listen :8080 --tun tun0
```

### Client-side configuration

Routing configuration:

```bash
sudo ip tuntap add dev tun0 mode tun
sudo ip addr add 192.168.99.1/24 dev tun0
sudo ip link set tun0 up
sudo ip route add default via 192.168.99.1 dev tun0 metric 100
```

Start the client:

```bash
sudo ./bin/twizcli --proxy 8.6.112.0:8080 --tun tun0 --outbound-iface wlp2s0
```