package keeper

import (
	"context"
	"encoding/binary"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ferux/flightcontrolcenter/internal/keeper/talk"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
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

type Conn struct {
	conn net.Conn

	deviceID uint64 // move to struct
	reqIDSeq func() uint64

	readMu sync.Mutex
	sendMu sync.Mutex
}

func WrapConnection(conn net.Conn) *Conn {
	return &Conn{
		conn:     conn,
		reqIDSeq: idSeqGenerator(),
	}
}

// Close connection.
func (c *Conn) Close() error { return c.conn.Close() }

func (c *Conn) extendConnectionLife() {
	_ = c.conn.SetDeadline(time.Now().Add(appendDeadline))
}

func (c *Conn) Send(ctx context.Context, msgType talk.MessageType, data []byte) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	rid := c.reqIDSeq()
	log.Printf("request_id=%d device_id=%d msg_type=%s body_size=%d sending", rid, c.deviceID, msgType, len(data))

	header := make([]byte, headerSize)
	binary.LittleEndian.PutUint64(header[0:8], rid)
	binary.LittleEndian.PutUint64(header[8:16], uint64(msgType))
	binary.LittleEndian.PutUint64(header[16:24], uint64(len(data)))

	_, err := c.conn.Write(header)
	if err != nil {
		return errors.Wrap(err, "unable to write header bytes")
	}

	_, err = c.conn.Write(data)
	if err != nil {
		return errors.Wrap(err, "unable to write data bytes")
	}

	c.extendConnectionLife()

	return nil
}

func (c *Conn) Read() (Packet, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	header := make([]byte, headerSize)
	n, err := io.ReadFull(c.conn, header)
	if err != nil {
		return Packet{}, errors.Wrap(err, "unable to read header")
	}
	if n != headerSize {
		// TODO: custom error type
		return Packet{}, errors.New("header data corrupted")
	}

	rid := binary.LittleEndian.Uint64(header[0:8])
	msgType := talk.MessageType(binary.LittleEndian.Uint64(header[8:16]))
	bodySize := binary.LittleEndian.Uint64(header[16:24])

	log.Printf("rid=%d device_id=%d msg_type=%s body_size=%d accepted message", rid, c.deviceID, msgType, bodySize)

	body := make([]byte, bodySize)
	n, err = io.ReadFull(c.conn, body)
	if err != nil {
		return Packet{}, errors.Wrap(err, "unable to read body")
	}

	if uint64(n) != bodySize {
		// TODO: custom error type
		return Packet{}, errors.New("body data corrupted")
	}

	c.extendConnectionLife()

	return Packet{
		Header: Header{
			RequestID:   rid,
			MessageType: msgType,
			BodySize:    bodySize,
		},
		Body: body,
	}, nil
}

func (c *Conn) DenyConnection(ctx context.Context, reason string, soft bool) error {
	msg := talk.Denied{
		Reason: reason,
		Soft:   soft,
	}

	data, err := proto.Marshal(&msg)
	if err != nil {
		errors.Wrap(err, "unable to marshal deny message")
	}

	msgType := talk.MessageType_DENY

	err = c.Send(ctx, msgType, data)
	if err != nil {
		log.Println("unable to send message: %v", err)
	}

	return c.Close()
}
