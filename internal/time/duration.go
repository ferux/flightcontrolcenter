package time

import (
	"encoding/json"
	"time"
)

type Duration time.Duration

func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(&d)
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var dText string
	err := json.Unmarshal(data, &dText)
	if err != nil {
		return err
	}

	dt, err := time.ParseDuration(dText)
	if err != nil {
		return err
	}

	*d = Duration(dt)
	return nil
}

func (d *Duration) String() string {
	return time.Duration(*d).String()
}
