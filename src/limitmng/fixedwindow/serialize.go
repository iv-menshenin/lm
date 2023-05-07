package fixedwindow

import (
	"encoding/json"
	"time"
)

const version = "1.0"

type fixedWindowJSON struct {
	Version   string        `json:"version"`
	Threshold float64       `json:"threshold"`
	Counter   float64       `json:"counter"`
	WindowSz  time.Duration `json:"windowSz"`
	Last      int64         `json:"last"`
}

func (b *FixedWindow) MarshalJSON() ([]byte, error) {
	b.mux.RLock()
	defer b.mux.RUnlock()

	var j = fixedWindowJSON{
		Version:   version,
		Threshold: b.threshold,
		Counter:   b.counter,
		WindowSz:  b.windowSz,
		Last:      b.last.UnixMilli(),
	}
	return json.Marshal(j)
}

func (b *FixedWindow) UnmarshalJSON(data []byte) error {
	var j fixedWindowJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*b = FixedWindow{
		threshold: j.Threshold,
		counter:   j.Counter,
		windowSz:  j.WindowSz,
		last:      time.UnixMilli(j.Last).UTC(),
	}
	return nil
}
