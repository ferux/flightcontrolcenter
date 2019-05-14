package yandex

import (
	"time"

	"github.com/ferux/yandexmapclient"
)

// StopInfo model
type StopInfo struct {
	IncomingTransport []TransportInfo
}

// TransportInfo model
type TransportInfo struct {
	Name   string
	Arrive time.Time
}

type client struct {
	c *yandexmapclient.Client
}
