package model

import "encoding/json"

type ServiceError struct {
	Message   string `json:"message,omitempty"`
	RequestID string `json:"request_id,omitempty"`

	Code int `json:"-"`
}

func (err ServiceError) Error() string {
	data, _ := json.Marshal(&err)
	return string(data)
}
