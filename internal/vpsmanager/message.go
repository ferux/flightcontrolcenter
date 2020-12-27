package vpsmanager

import (
	"encoding/json"
	"fmt"
	"strings"
)

type serviceResponse struct {
	Status    string          `json:"status"`
	StatusMsg string          `json:"status_msg"`
	Data      json.RawMessage `json:"data"`
}

func parseResponseData(data []byte) (sr serviceResponse, err error) {
	err = json.Unmarshal(data, &sr)
	if err != nil {
		return sr, fmt.Errorf("unmarshalling json: %w", err)
	}

	return sr, nil
}

// unmarshalData unmarshals data to destination object.
func (s serviceResponse) unmarshalData(dst interface{}) (err error) {
	return json.Unmarshal(s.Data, dst)
}

func (s serviceResponse) isOK() bool {
	return strings.EqualFold(s.Status, "ok")
}

type dnsInfo struct {
	ID         int64       `json:"id"`
	Name       string      `json:"name"`
	FullName   string      `json:"full_name"`
	Created    string      `json:"created"`
	Updated    string      `json:"updated"`
	End        string      `json:"end"`
	Status     string      `json:"status"`
	StatusText *string     `json:"status_text"`
	Real       bool        `json:"real"`
	Can        modifyFlags `json:"can"`
}

type dnsRecord struct {
	ID        int64       `json:"id"`
	Host      string      `json:"host"`
	Type      string      `json:"type"`
	Value     string      `json:"value"`
	Timestamp string      `json:"timestamp"`
	Can       modifyFlags `json:"can"`
}

type modifyFlags struct {
	Update bool `json:"update"`
	Delete bool `json:"delete"`
}
