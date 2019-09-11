package device

import (
	"net"
	"time"

	"github.com/ferux/flightcontrolcenter/internal/model"
)

// Connection links connection and device.
type Connection struct {
	device *model.Device
	conn   net.Conn
}

// GetDevice returns assocciated device.
func (c *Connection) GetDevice() *model.Device {
	return c.device
}

// IsRegistered show the device has been authenticated and active.
func (c *Connection) IsRegistered() bool {
	return c.device != nil
}

// Write to device's connection.
func (c *Connection) Write(data []byte) (int, error) {
	return c.conn.Write(data)
}

func (c *Connection) Read(data []byte) (int, error) {
	return c.conn.Read(data)
}

// Close device's connection.
func (c *Connection) Close() error {
	return c.conn.Close()
}

// SetDeadline of the connection.
func (c *Connection) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline of the connection.
func (c *Connection) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline of the connecton.
func (c *Connection) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// RemoteAddr of connection
func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
