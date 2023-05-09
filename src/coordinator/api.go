package coordinator

import (
	"errors"
	"sync/atomic"
)

const Mine = "MINE"

func (m *Manager) CheckKey(key string) (string, error) {
	if atomic.LoadInt64(&m.armed) != Armed {
		return "", errors.New("not ready")
	}
	if len(key) > 128 {
		return "", errors.New("too long key")
	}
	if instance := m.ins.search(key); instance != "" {
		return instance, nil
	}
	errCh := m.awaitResponses(cmdSaved, []byte(key))
	if err := m.advertiseThatIsMine(key); err != nil {
		return "", err
	}
	if err := <-errCh; err != nil {
		return "", err
	}
	return Mine, nil
}
