package yandex

import (
	"context"
	"time"

	"github.com/ferux/flightcontrolcenter/internal/logger"
	"github.com/ferux/yandexmapclient"
)

// Client for interacting with yandex client
type Client interface {
	Fetch(ctx context.Context, stopID string, prognosis bool) (StopInfo, error)
	UpdateToken() error
}

// New initialisates new client. It's acceptable to pass nil for logger
// nolint:interfacer
func New(l logger.Logger) (Client, error) {
	c, err := yandexmapclient.New(yandexmapclient.WithLogger(l))
	if err != nil {
		return nil, err
	}

	return &client{c: c}, nil
}

func (c *client) Fetch(ctx context.Context, stopID string, prognosis bool) (StopInfo, error) {
	si, err := c.c.FetchStopInfo(ctx, stopID, prognosis)
	if err != nil {
		return StopInfo{}, err
	}

	if si.Data == nil {
		return StopInfo{}, nil
	}

	var s = StopInfo{IncomingTransport: make([]TransportInfo, 0, len(si.Data.Properties.StopMetaData.Transport))}

	if len(si.Data.Properties.StopMetaData.Transport) == 0 {
		return StopInfo{}, nil
	}

	for _, tr := range si.Data.Properties.StopMetaData.Transport {
		ti := extractTransportInfo(tr)
		if time.Now().After(ti.Arrive) {
			continue
		}

		s.IncomingTransport = append(s.IncomingTransport, ti)
	}

	return s, nil
}

func extractTransportInfo(tr yandexmapclient.TransportInfo) TransportInfo {
	var ti = TransportInfo{Name: tr.Name}

	if len(tr.Threads) == 0 {
		return TransportInfo{}
	}

	bs := tr.Threads[0]
	if len(bs.BriefSchedule.Events) > 0 {
		if !bs.BriefSchedule.Events[0].Scheduled.Time.IsZero() {
			ti.Arrive = bs.BriefSchedule.Events[0].Scheduled.Time
			ti.Method = "scheduled (best)"
		} else {
			ti.Arrive = bs.BriefSchedule.Events[0].Estimated.Time
			ti.Method = "estimated (good)"
		}

		return ti
	}

	if time.Now().After(bs.BriefSchedule.Frequency.End.Time) {
		ti.Method = "end (worst)"
		ti.Arrive = bs.BriefSchedule.Frequency.Begin.Time
		return ti
	}

	ti.Method = "frequency (so-so)"
	ti.Arrive = time.Now().Add(time.Second * time.Duration(bs.BriefSchedule.Frequency.Value))
	return ti
}

func (c *client) UpdateToken() error { return c.c.UpdateToken() }
