package keeper

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ferux/flightcontrolcenter/internal/keeper/talk"
	"github.com/ferux/flightcontrolcenter/internal/model"

	"github.com/gogo/protobuf/proto"
)

func idSeqGenerator() func() uint64 {
	var currentID uint64

	return func() uint64 {
		return atomic.AddUint64(&currentID, 1)
	}
}

// Packet is a generic message for communication
type Packet struct {
	Header Header
	Body   []byte
}

// Header should contain request_id and body_size. Otherwise packet will be dropped
type Header struct {
	RequestID   uint64
	MessageType talk.MessageType
	BodySize    uint64
}

const headerSize = 8 + 8 + 8

const appendDeadline = time.Second * 60

// Conn stores meta information about single connection.
type Conn struct {
	conn net.Conn

	device   model.Device
	reqIDSeq func() uint64
	readMu   sync.Mutex
	sendMu   sync.Mutex
}

const supportedMajor = 1

// Handshale connection with other device. It doesn't
func HandshakeConnection(timeout time.Duration, conn net.Conn, r Repo) (c *Conn, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c = &Conn{
		conn:     conn,
		reqIDSeq: idSeqGenerator(),
	}

	p, err := c.Read()

	if err != nil {
		return nil, fmt.Errorf("reading from connection: %w", err)
	}

	if p.Header.MessageType != talk.MessageType_CLIENT_INFO {
		return nil, ErrUnexpectedPacket
	}

	var msg talk.ClientInfo
	if err = proto.Unmarshal(p.Body, &msg); err != nil {
		return nil, fmt.Errorf("unmarshalling message: %w", err)
	}

	v := msg.GetAPIVersion()
	if v.GetMajor() != supportedMajor {
		return nil, ErrUnsupportedVersion
	}

	device, err := r.GetByUUID(ctx, msg.GetDeviceID())

	var nf NotFoundError

	switch {
	case errors.As(err, &nf):
		device = deviceFromClientInfo(msg)
		device.CreatedAt = time.Now()
		device.UpdatedAt = time.Now()
	case err == nil:
		device.Version = msg.GetAPIVersion().String()
	default:
		return nil, fmt.Errorf("getting device from repo: %w", err)
	}

	device.IP = conn.RemoteAddr().String()
	device.State = model.DeviceStateOnline
	device.StateFixAt = time.Now()

	if device.ID == 0 {
		id, err := r.Insert(ctx, device)

		if err != nil {
			return nil, fmt.Errorf("inserting device: %w", err)
		}

		device.ID = id
	} else {
		err = r.Update(ctx, device)
		if err != nil {
			return nil, fmt.Errorf("updating device: %w", err)
		}
	}

	c.device = device

	return c, nil
}

func deviceFromClientInfo(c talk.ClientInfo) model.Device {
	return model.Device{
		Type:    model.DeviceType(c.GetDeviceType()),
		Token:   c.GetSecret(),
		MAC:     c.GetMAC(),
		Version: c.GetAPIVersion().String(),
	}
}

// Close connection.
func (c *Conn) Close() error { return c.conn.Close() }

func (c *Conn) extendConnectionLife() error {
	return c.conn.SetDeadline(time.Now().Add(appendDeadline))
}

// Send message to device.
func (c *Conn) Send(ctx context.Context, msgType talk.MessageType, msg proto.Message) (err error) {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshalling protobuf: %w", err)
	}

	rid := c.reqIDSeq()
	log.Printf("request_id=%d device_id=%d msg_type=%s body_size=%d sending", rid, c.device.ID, msgType, len(data))

	header := make([]byte, headerSize)
	binary.LittleEndian.PutUint64(header[0:8], rid)
	binary.LittleEndian.PutUint64(header[8:16], uint64(msgType))
	binary.LittleEndian.PutUint64(header[16:24], uint64(len(data)))

	_, err = c.conn.Write(header)
	if err != nil {
		return fmt.Errorf("writing header bytes: %w", err)
	}

	_, err = c.conn.Write(data)
	if err != nil {
		return fmt.Errorf("writing data bytes: %w", err)
	}

	err = c.extendConnectionLife()
	if err != nil {
		return fmt.Errorf("extending deadline: %w", err)
	}

	return nil
}

// Read incoming message from connection.
func (c *Conn) Read() (Packet, error) {
	// mutex protects from being read sequence of messages in another place.
	c.readMu.Lock()
	defer c.readMu.Unlock()

	header := make([]byte, headerSize)
	n, err := io.ReadFull(c.conn, header)

	if err != nil {
		return Packet{}, fmt.Errorf("reading header: %w", err)
	}

	if n != headerSize {
		return Packet{}, PermamentError("header data corrupted")
	}

	rid := binary.LittleEndian.Uint64(header[0:8])
	msgType := talk.MessageType(binary.LittleEndian.Uint64(header[8:16]))
	bodySize := binary.LittleEndian.Uint64(header[16:24])

	log.Printf("rid=%d device_id=%d msg_type=%s body_size=%d accepted message", rid, c.device.ID, msgType, bodySize)

	body := make([]byte, bodySize)
	n, err = io.ReadFull(c.conn, body)

	if err != nil {
		return Packet{}, fmt.Errorf("reading body: %w", err)
	}

	if uint64(n) != bodySize {
		return Packet{}, PermamentError("body data corrupted")
	}

	err = c.extendConnectionLife()
	if err != nil {
		return Packet{}, fmt.Errorf("extending deadline: %w", err)
	}

	return Packet{
		Header: Header{
			RequestID:   rid,
			MessageType: msgType,
			BodySize:    bodySize,
		},
		Body: body,
	}, nil
}

// DenyConnection writes reason and closes connection. It's okay to call this method
// if Conn is null or underlying connection is null or closed.
func (c *Conn) DenyConnection(ctx context.Context, reason string, soft bool) (err error) {
	if c == nil || c.conn == nil {
		return nil
	}

	msgType := talk.MessageType_DENY
	msg := talk.Denied{
		Reason: reason,
		Soft:   soft,
	}

	err = c.Send(ctx, msgType, &msg)
	if err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	err = c.Close()
	if err != nil {
		return fmt.Errorf("closing connetion: %w", err)
	}

	return nil
}
