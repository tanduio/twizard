# twizard

A lightweight TUN-based forwarding proxy designed mainly for educational purposes. This project explores the fundamentals of VPN-like packet forwarding through a simple, transparent proxy implementation.

> **⚠️  Experimental Status**  
> This is primarily a learning project focused on understanding low-level networking concepts. While functional for basic forwarding, it's not recommended for production use. Expect bugs and missing features.

## Overview

**twizard** implements a simple TUN interface proxy that forwards network traffic between client and server. The current implementation provides basic TCP forwarding as a foundation for exploring more advanced VPN concepts.

## Current Status & Roadmap

### ✅ Implemented
- Basic TUN interface setup
- Simple TCP packet forwardinga
- Server-side packet reflection

### 🔄 Planned Features
- [ ] Complete direct flow with proper server-side processing
- [ ] UDP forwarding (to mitigate TCP meltdown issues)
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
