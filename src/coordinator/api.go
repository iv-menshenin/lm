package coordinator

import (
	"context"
	"errors"
	"math/rand"
	"sync/atomic"
	"time"
)

const Mine = "MINE"

func (m *Manager) CheckKey(ctx context.Context, key string) (string, error) {
	if atomic.LoadInt64(&m.armed) != Armed {
		return "", errors.New("not ready")
	}
	if len(key) > 128 {
		return "", errors.New("too long key")
	}
	for {
		if instance := m.ins.search(key); instance != "" {
			return instance, nil
		}
		owned, err := m.tryToOwn(key)
		if err != nil {
			return "", err
		}
		if owned {
			return Mine, nil
		}
		if instance := m.ins.search(key); instance != "" {
			return instance, nil
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()

		case <-time.After(time.Duration(rand.Intn(50)+50) * time.Millisecond):
			// next trying
		}
	}
}
