package ping

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

type Store interface {
	Ping(msg Message)
	Subscribe(fn NotifyDeviceStateChanged)
	GetDevices() []*Device
}

type NotifyDeviceStateChanged func(d *Device)

type store struct {
	devices map[string]*Device
	mu      sync.RWMutex
	subs    []NotifyDeviceStateChanged
	logger  zerolog.Logger
	sentry  *sentry.Client
}

func New(sentry *sentry.Client) Store {
	s := &store{
		sentry: sentry,

		devices: make(map[string]*Device),
		logger:  zerolog.New(os.Stdout).With().Timestamp().Logger(),
	}

	go s.start()

	return s
}

func (c *store) Subscribe(fn NotifyDeviceStateChanged) {
	c.subs = append(c.subs, fn)
}

func (c *store) notify(d *Device) {
	if len(c.subs) == 0 {
		return
	}

	for _, fn := range c.subs {
		fn(d)
	}
}

func (c *store) start() {
	delay := time.Second * 60
	for {
		c.updateDevicesState()
		time.Sleep(delay)
	}
}

// TODO: possibly might be very laggy if there will be a lot of devices. Copy map and procced it, maybe?
func (c *store) updateDevicesState() {
	now := time.Now()
	c.mu.Lock()
	if lockTime := time.Since(now); lockTime > time.Second*3 {
		c.logger.Error().Dur("lock_time", lockTime).Msg("took too much time")
		c.sentry.CaptureException(
			errors.New("lock_time took too long"),
			&sentry.EventHint{
				Data: map[string]interface{}{
					"lock_time": lockTime,
				},
			},
			sentry.NewScope(),
		)
	}

	defer c.mu.Unlock()

	for k := range c.devices {
		device := c.devices[k]
		if time.Since(device.PingedAt) > time.Minute && device.IsOnline {
			c.logger.Debug().Str("device", k).Msg("went offline")
			device.IsOnline = false
			c.notify(device)
		}
	}
}

// Ping proceeds ping message from device and updates its state
func (c *store) Ping(m Message) {
	name := m.Name + "@" + m.IP
	c.mu.RLock()
	device, ok := c.devices[name]
	c.mu.RUnlock()

	now := time.Now()
	if !ok {
		device = &Device{
			Message:   m,
			IsOnline:  true,
			CreatedAt: now,
			UpdatedAt: now,
			PingedAt:  now,
		}

		c.mu.Lock()
		c.devices[name] = device
		c.mu.Unlock()

		c.logger.Debug().Str("device", name).Msg("registered")
		c.notify(device)
		return
	}

	if device.Type != m.Type {
		c.logger.Warn().Interface("origin", device).Interface("new", m).Msg("device type is different, skiping")
		return
	}

	device.PingedAt = now
	if device.Revision != m.Revision {
		device.Revision = m.Revision
		device.UpdatedAt = now
	}

	if device.Branch != m.Branch {
		device.Branch = m.Revision
		device.UpdatedAt = now
	}

	if !device.IsOnline {
		c.logger.Debug().Str("device", name).Msg("came back online")
		device.IsOnline = true
		c.notify(device)
	}
}

// GetDevices gets all stored devices
func (c *store) GetDevices() []*Device {
	if len(c.devices) == 0 {
		return []*Device{}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	devices := make([]*Device, 0, len(c.devices))
	for k := range c.devices {
		device := c.devices[k]
		devices = append(devices, device)
	}
	return devices
}
