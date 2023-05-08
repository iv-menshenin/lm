package net

import (
	"fmt"
	"net"
)

type (
	Listener struct {
		port uint16
		addr []net.IP
		pc   net.PacketConn
	}
)

const DefaultPort = 7999

func New(port uint16) (*Listener, error) {
	if port == 0 {
		port = DefaultPort
	}
	pc, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	var l = Listener{
		port: port,
		pc:   pc,
	}
	return &l, nil
}

func (l *Listener) Listen() error {
	var buf = make([]byte, 1024)
	n, addr, err := l.pc.ReadFrom(buf)
	if err != nil {
		return err
	}
	fmt.Printf("received: %d from %s: %s", n, addr.String(), string(buf[:n]))
	return nil
}

func (l *Listener) Send() error {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("192.168.1.255:%d", l.port))
	if err != nil {
		return err
	}
	_, err = l.pc.WriteTo([]byte("data to transmit"), addr)
	if err != nil {
		return err
	}
	return nil
}

func (l *Listener) Close() error {
	return l.pc.Close()
}

func (l *Listener) discoverSubnets() error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	l.addr = make([]net.IP, 0, len(ifaces))
	for _, i := range ifaces {
		addrs, err := i.MulticastAddrs()
		if err != nil {
			return err
		}
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				l.addr = append(l.addr, v.IP)
			case *net.IPAddr:
				l.addr = append(l.addr, v.IP)
			}
		}
	}
	return nil
}
