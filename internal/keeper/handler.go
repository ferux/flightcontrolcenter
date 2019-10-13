package keeper

import (
	"context"
	"log"

	"github.com/ferux/flightcontrolcenter/internal/keeper/talk"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

type KeeperError string

func (err KeeperError) Error() string { return string(err) }

const currentAPIVersion uint64 = 1

// handler serves incoming message
type handler func(context.Context, []byte, *Conn) error

func handleClientInfo(ctx context.Context, data []byte, conn *Conn) error {
	var msg = talk.ClientInfo{}
	err := proto.Unmarshal(data, &msg)
	if err != nil {
		return errors.Wrap(err, "unable to unmarshal ClientInfo message")
	}

	if len(msg.DeviceId) == 0 {
		// device is new
		log.Println("new device, hooray!")
		return nil
	}

	if msg.ApiVersion.Major < currentAPIVersion {
		// we should deny connection
		return conn.DenyConnection(ctx, "version not supported", false)
	}

	return nil
}
