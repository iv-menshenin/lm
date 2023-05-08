package coordinator

import (
	"errors"
	"net"

	"github.com/iv-menshenin/lm/transport"
)

func (m *Manager) sendKnockKnock() error {
	var data = make([]byte, 0, 1024)
	data = append(data, cmdBroadKnock...)
	data = append(data, m.ID[:]...)
	return m.ls.SendAll(data)
}

func (m *Manager) sendWelcome(addr net.Addr) error {
	var data = make([]byte, 0, 1024)
	data = append(data, cmdWelcome...)
	data = append(data, m.ID[:]...)
	return m.ls.Send(data, addr)
}

func (m *Manager) advertiseThatIsMine(key string) error {
	var data = make([]byte, 0, 1024)
	data = append(data, cmdBroadMine...)
	data = append(data, m.ID[:]...)
	data = append(data, key...)
	return m.ls.SendAll(data)
}

func (m *Manager) sendSaved(addr net.Addr, key []byte) error {
	var data = make([]byte, 0, 1024)
	data = append(data, cmdSaved...)
	data = append(data, m.ID[:]...)
	data = append(data, key...)
	return m.ls.Send(data, addr)
}

func (m *Manager) sendReset(key []byte) error {
	var data = make([]byte, 0, 1024)
	data = append(data, cmdBroadReset...)
	data = append(data, m.ID[:]...)
	data = append(data, key...)
	return m.ls.SendAll(data)
}

func (m *Manager) sendCompareInstances(id []byte) error {
	var data = make([]byte, 0, 1024)
	data = append(data, cmdBroadCompare...)
	data = append(data, m.ID[:]...)
	data = append(data, id...)
	return m.ls.SendAll(data)
}

func (m *Manager) sendCompared(addr net.Addr, id []byte) error {
	var data = make([]byte, 0, 1024)
	data = append(data, cmdCompared...)
	data = append(data, m.ID[:]...)
	data = append(data, id...)
	return m.ls.Send(data, addr)
}

var (
	cmdBroadKnock   = []byte("KNCK")
	cmdBroadMine    = []byte("MINE")
	cmdBroadReset   = []byte("RSET")
	cmdBroadCompare = []byte("CMPI")

	cmdWelcome  = []byte("WLCM")
	cmdSaved    = []byte("SAVD")
	cmdCompared = []byte("CMPO")
)

type Msg struct {
	cmd    []byte
	sender []byte
	data   []byte
	addr   net.Addr
}

func parse(rcv *transport.Received) (Msg, error) {
	if len(rcv.Data) < 20 {
		return Msg{}, errors.New("bad message")
	}
	msg := Msg{
		cmd:    rcv.Data[0:4],
		sender: rcv.Data[4:20],
		data:   rcv.Data[20:],
		addr:   rcv.Addr,
	}
	return msg, nil
}
