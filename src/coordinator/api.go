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
	if !m.ins.toOwn(key) {
		m.ins.fromOwn(key)
		return "", errors.New("can't own, race")
	}
	var trys = 5
	for {
		errCh := m.awaitResponses(cmdSaved, []byte(key))
		if err := m.advertiseThatIsMine(key); err != nil {
			m.ins.fromOwn(key)
			return "", err
		}
		if err := <-errCh; err != nil {
			if trys--; trys > 0 {
				continue
			}
			m.ins.fromOwn(key)
			return "", err
		}
		return Mine, nil
	}
}
