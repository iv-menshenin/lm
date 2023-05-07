package leakybucket

import (
	"encoding/json"
	"time"
)

const version = "1.0"

type leakyBucketJSON struct {
	Version   string        `json:"version"`
	Threshold float64       `json:"threshold"`
	Counter   float64       `json:"counter"`
	WindowSz  time.Duration `json:"windowSz"`
	Bandwidth float64       `json:"bandwidth"`
	Last      int64         `json:"last"`
}

func (b *LeakyBucket) MarshalJSON() ([]byte, error) {
	b.mux.RLock()
	defer b.mux.RUnlock()

	var j = leakyBucketJSON{
		Version:   version,
		Threshold: b.threshold,
		Counter:   b.counter,
		WindowSz:  b.windowSz,
		Bandwidth: b.bandwidth,
		Last:      b.last.UnixMilli(),
	}
	return json.Marshal(j)
}

func (b *LeakyBucket) UnmarshalJSON(data []byte) error {
	var j leakyBucketJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*b = LeakyBucket{
		threshold: j.Threshold,
		counter:   j.Counter,
		windowSz:  j.WindowSz,
		bandwidth: j.Bandwidth,
		last:      time.UnixMilli(j.Last).UTC(),
	}
	return nil
}
