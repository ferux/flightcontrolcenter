package model

type Device struct {
	ID      DeviceID   `json:"id"`
	Type    DeviceType `json:"type"`
	Name    string     `json:"name"`
	Token   string     `json:"token"`
	Version string     `json:"version"`
}

type DeviceID uint64

type DeviceType uint64

const (
	DeviceUnknown DeviceType = iota
	DeviceChip
)
