package ping

import "time"

// Message stores information about device
type Message struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Revision string `json:"revision,omitempty"`
	Branch   string `json:"branch,omitempty"`

	IP string `json:"ip,omitempty"`
}

type Device struct {
	Message

	IsOnline  bool      `json:"is_online"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	PingedAt  time.Time `json:"pinged_at"`
}
