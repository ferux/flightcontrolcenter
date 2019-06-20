package api

type ServiceError struct {
	Message   string `json:"message,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

type StopInfo struct {
	Name      string `json:"name,omitempty"`
	Next      string `json:"next,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}
