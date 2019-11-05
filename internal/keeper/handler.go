package keeper

import (
	"context"
	"fmt"

	"github.com/ferux/flightcontrolcenter/internal/keeper/talk"

	"github.com/gogo/protobuf/proto"
)

// handler serves incoming message
type handler func(ctx context.Context, msgData []byte, conn *Conn) error

func handlePong(_ context.Context, msgData []byte, conn *Conn) (err error) {
	var msg talk.Pong

	err = proto.Unmarshal(msgData, &msg)
	if err != nil {
		return fmt.Errorf("unmarshalling data: %w", err)
	}

	err = conn.extendConnectionLife()
	if err != nil {
		return fmt.Errorf("extending connection life: %w", err)
	}

	return nil
}
