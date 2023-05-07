package tokenbucket

import (
	"encoding/json"
	"time"
)

const version = "1.0"

type tokenBucketJSON struct {
	Version   string        `json:"version"`
	Threshold float64       `json:"threshold"`
	Counter   float64       `json:"counter"`
	WindowSz  time.Duration `json:"windowSz"`
	Bandwidth float64       `json:"bandwidth"`
	Last      int64         `json:"last"`
}

func (b *TokenBucket) MarshalJSON() ([]byte, error) {
	b.mux.RLock()
	defer b.mux.RUnlock()

	var j = tokenBucketJSON{
		Version:   version,
		Threshold: b.threshold,
		Counter:   b.counter,
		WindowSz:  b.windowSz,
		Bandwidth: b.bandwidth,
		Last:      b.last.UnixMilli(),
	}
	return json.Marshal(j)
}

func (b *TokenBucket) UnmarshalJSON(data []byte) error {
	var j tokenBucketJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*b = TokenBucket{
		threshold: j.Threshold,
		counter:   j.Counter,
		windowSz:  j.WindowSz,
		bandwidth: j.Bandwidth,
		last:      time.UnixMilli(j.Last).UTC(),
	}
	return nil
}
