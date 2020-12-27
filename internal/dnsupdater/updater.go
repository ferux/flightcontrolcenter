package dnsupdater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/ferux/flightcontrolcenter/internal/config"
	"github.com/ferux/flightcontrolcenter/internal/fcontext"
	"github.com/ferux/flightcontrolcenter/internal/model"
)

type Client interface {
	UpdateDNS(ctx context.Context, namespace, ip string) (err error)
}

type clientNoop struct{}

func (clientNoop) UpdateDNS(_ context.Context, _, _ string) error {
	return nil
}

type client struct {
	addr   string
	secret string

	namespaces map[string][]string
}

func New(_ context.Context, cfg config.DNSUpdater) Client {
	if len(cfg.Namespaces) == 0 {
		return clientNoop{}
	}

	return client{
		addr:       cfg.Address,
		namespaces: cfg.Namespaces,
		secret:     cfg.Secret,
	}
}

type response struct {
	Success  bool     `json:"Success"`
	Message  string   `json:"Message"`
	Domain   string   `json:"Domain"`
	Domains  []string `json:"Domains"`
	Address  string   `json:"Address"`
	AddrType string   `json:"AddrType"`
}

// Update implements client interface.
func (c client) UpdateDNS(ctx context.Context, namespace, ip string) (err error) {
	var requrl *url.URL

	requrl, err = url.Parse(c.addr + "/update")
	if err != nil {
		return fmt.Errorf("parsing url addr: %w", err)
	}

	names, ok := c.namespaces[namespace]
	if !ok {
		return model.ErrNotFound
	}

	joinedNames := strings.Join(names, ",")
	err = updateRecord(ctx, requrl, c.secret, joinedNames, ip)
	if err != nil {
		return fmt.Errorf("updating %s: %w", joinedNames, err)
	}

	return nil
}

func updateRecord(ctx context.Context, requrl *url.URL, secret, name, ip string) (err error) {
	log := fcontext.Logger(ctx)
	q := requrl.Query()
	q.Set("secret", secret)
	q.Set("domain", name)
	q.Set("addr", ip)

	requrl.RawQuery = q.Encode()

	log.Debugf("sending request to %s", requrl.Hostname())

	var req *http.Request

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, requrl.String(), nil)
	if err != nil {
		return fmt.Errorf("making new request: %w", err)
	}

	req.Header.Set("X-Request-ID", fcontext.RequestID(ctx))

	var resp *http.Response

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("proceeding request: %w", err)
	}

	defer func() {
		var errdefer error

		_, errdefer = io.Copy(ioutil.Discard, resp.Body)
		if errdefer != nil {
			log.Errf("discarding body leftovers: %v", err)
		}

		errdefer = resp.Body.Close()
		if errdefer != nil {
			log.Errf("closing response body: %v", err)
		}
	}()

	var respMessage response

	err = json.NewDecoder(resp.Body).Decode(&respMessage)
	if err != nil {
		return fmt.Errorf("decoding body: %w", err)
	}

	log.Debugf("response from dns: %+v", respMessage)

	if !respMessage.Success {
		return fmt.Errorf("updating dns error: %w", model.Error(respMessage.Message))
	}

	return nil
}
