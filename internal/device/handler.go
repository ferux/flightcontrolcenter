package device

import (
	"bytes"
	"context"
	"encoding/gob"
	"time"

	"github.com/ferux/flightcontrolcenter/internal/fcontext"

	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

type HelloRequest struct {
	RequestID     string
	ServerVersion string
	Timestamp     time.Time
}

func (c *Connection) sendHelloRequest(ctx context.Context, request HelloRequest) error {
	logger := fcontext.Logger(ctx)

	request.Timestamp = time.Now().UTC()
	request.RequestID = uuid.New()

	logger.Debugf("sending HelloRequest message: %v", request)

	err := gob.NewEncoder(c).Encode(&request)
	if err != nil {
		return errors.Wrap(err, "unable to send hello request")
	}

	return nil
}

type helloResponse struct {
	RequestID     string
	ClientVersion string
	Timestamp     time.Time
}

func (a *API) handleHelloResponse(ctx context.Context, msgData []byte) error {
	logger := fcontext.Logger(ctx)
	deviceID := fcontext.DeviceID(ctx)
	logger.Debugf("handling hello response from %d", deviceID)

	var msg helloResponse
	err := gob.NewDecoder(bytes.NewReader(msgData)).Decode(&msg)
	if err != nil {
		return errors.Wrap(err, "unable to decode data")
	}

	logger.Debugf("message %v", msg)

	return nil
}
