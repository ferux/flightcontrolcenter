package model

import "time"

// Device is an object connected to keeper service.
type Device struct {
	ID      DeviceID   `json:"id"`
	Type    DeviceType `json:"type"`
	Name    string     `json:"name"`
	Token   string     `json:"token"`
	Version string     `json:"version"`
	MAC     string     `json:"mac"`
	IP      string     `json:"ip"`

	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
	StateFixAt time.Time   `json:"state_fix_at"`
	State      DeviceState `json:"state"`

	// this id is used to identificate device that
	// came via tcp connection.
	UUID string `json:"-"`
}

// DeviceID is a typed uint64.
type DeviceID uint64

// DeviceType describes device type.
type DeviceType uint64

const (
	DeviceUnknown DeviceType = iota
	DevicePhone
	DeviceMedia
	DevicePC
)

type DeviceState uint8

const (
	DeviceStateUnknown DeviceState = iota
	DeviceStateOnline
	DeviceStateOffline
	DeviceStateBanned
	DeviceStateDeleted
)
