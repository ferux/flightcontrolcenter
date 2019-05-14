package yandex

import (
	"time"

	"github.com/ferux/flightcontrolcenter/internal/logger"

	"github.com/ferux/yandexmapclient"
)

// Client for interacting with yandex client
type Client interface {
	Fetch(stopID string) (StopInfo, error)
	UpdateToken() error
}

// New initialisates new client. It's acceptable to pass nil for logger
func New(l logger.Logger) (Client, error) {
	c, err := yandexmapclient.New(yandexmapclient.WithLogger(l))
	if err != nil {
		return nil, err
	}

	return &client{c: c}, nil
}

func (c *client) Fetch(stopID string) (StopInfo, error) {
	si, err := c.c.FetchStopInfo(stopID)
	if err != nil {
		return StopInfo{}, err
	}

	var s = StopInfo{IncomingTransport: make([]TransportInfo, 0, len(si.Data.Properties.StopMetaData.Transport))}
	var now = time.Now()
	for _, tr := range si.Data.Properties.StopMetaData.Transport {
		t, err := time.Parse("15:04", tr.BriefSchedule.DepartureTime)
		if err != nil {
			continue
		}
		t.AddDate(now.Year(), int(now.Month()-1), now.Day())
		s.IncomingTransport = append(s.IncomingTransport, TransportInfo{Name: tr.Name, Arrive: t})
	}

	return s, nil
}

func (c *client) UpdateToken() error { return c.c.UpdateToken() }
