package udp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

type (
	Listener struct {
		port uint16
		addr []net.IP
		conn net.PacketConn
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
		conn: pc,
	}
	if err = l.discoverSubnets(); err != nil {
		return nil, err
	}
	return &l, nil
}

type Received struct {
	Addr net.Addr
	Data []byte
}

func (l *Listener) Listen() (*Received, error) {
	var buf = make([]byte, 1024)
	n, addr, err := l.conn.ReadFrom(buf)
	if err != nil {
		return nil, err
	}
	received := Received{
		Addr: addr,
		Data: buf[:n],
	}
	return &received, nil
}

func (l *Listener) Send(data []byte) error {
	for _, addr := range l.addr {
		udp := net.UDPAddr{
			IP:   addr,
			Port: int(l.port),
		}
		if _, err := l.conn.WriteTo(data, &udp); err != nil {
			return err
		}
	}
	return nil
}

func (l *Listener) Close() error {
	return l.conn.Close()
}

func (l *Listener) discoverSubnets() error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	l.addr = make([]net.IP, 0, len(ifaces))
	for _, i := range ifaces {
		if i.Name == "lo" {
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			return err
		}
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ipV4 := v.IP.To4()
				if ipV4 == nil {
					continue
				}
				ip := make(net.IP, len(ipV4))
				binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(ipV4)|^binary.BigEndian.Uint32(net.IP(v.Mask).To4()))
				l.addr = append(l.addr, ip)
			}
		}
	}
	return nil
}

func lastAddr(n *net.IPNet) (net.IP, error) { // works when the n is a prefix, otherwise...
	if n.IP.To4() == nil {
		return net.IP{}, errors.New("does not support IPv6 addresses.")
	}
	ip := make(net.IP, len(n.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(n.IP.To4())|^binary.BigEndian.Uint32(net.IP(n.Mask).To4()))
	return ip, nil
}
