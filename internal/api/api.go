package api

type StopInfo struct {
	Name      string `json:"name,omitempty"`
	Next      string `json:"next,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}
