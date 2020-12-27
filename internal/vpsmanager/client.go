package vpsmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/ferux/flightcontrolcenter/internal/config"
	"github.com/ferux/flightcontrolcenter/internal/fcontext"
	"github.com/ferux/flightcontrolcenter/internal/model"
)

type Client interface {
	UpdateDNS(ctx context.Context, namespace string, ip string) (err error)
}

type HTTPClient struct {
	client *http.Client
	url    string
	token  string
	// TODO: remove once HTTPClient might be embbed into DNSUpdated client.
	namespaces map[string][]string

	prepared bool
}

// New created new VPSManager Client.
// TODO: merge with DNSUpdater for updating dns.
func New(ctx context.Context, cfg config.VPS, cfgDNSUpdated config.DNSUpdater) (c HTTPClient, err error) {
	if cfg.Token == "" && cfg.User == "" && cfg.Pass == "" {
		return HTTPClient{}, fmt.Errorf("either token or user and pass should be set: %w", model.ErrMissingParameter)
	}

	c = HTTPClient{
		client:     http.DefaultClient,
		url:        cfg.URL,
		token:      cfg.Token,
		namespaces: cfgDNSUpdated.Namespaces,
		prepared:   true,
	}

	if cfg.Token == "" {
		c.token, err = c.authorize(ctx, cfg.User, cfg.Pass)
		if err != nil {
			return HTTPClient{}, fmt.Errorf("getting token: %w", err)
		}
	} else {
		err = c.checkAvailability(ctx)
		if err != nil {
			return HTTPClient{}, fmt.Errorf("checking token: %w", err)
		}
	}

	return c, nil
}

func (c HTTPClient) checkAvailability(ctx context.Context) (err error) {
	const reqpath = "/v1/account"

	fcontext.Logger(ctx).Debug("checking token availability")

	_, err = c.performRequest(ctx, http.MethodGet, reqpath, nil)
	if err != nil {
		return err
	}

	return nil
}

// UpdateDNS implements Client interface.
func (c HTTPClient) UpdateDNS(ctx context.Context, namespace string, ip string) (err error) {
	var infos []dnsInfo

	infos, err = c.getDNSInfos(ctx)
	if err != nil {
		return fmt.Errorf("getting dns records: %w", err)
	}

	var info dnsInfo
	// TODO: handle more than 1 dns info.
	if len(infos) == 0 {
		return model.ErrNotFound
	}

	info = infos[0]

	var records []dnsRecord
	records, err = c.getDNSRecords(ctx, info)
	if err != nil {
		return fmt.Errorf("getting dns records: %w", err)
	}

	logger := fcontext.Logger(ctx)

	recordsIdxByValue := make(map[string]dnsRecord)
	for _, record := range records {
		if record.Type != "A" {
			logger.Debugf("skipping record host=%s type=%s reason='only type A allowed'", record.Host, record.Type)

			continue
		}

		logger.Debugf("indexing record %#v", record)

		recordsIdxByValue[record.Host] = record
	}

	var ok bool
	var record dnsRecord
	recordsFromNS := c.namespaces[namespace]

	for _, recordFromNS := range recordsFromNS {
		recordFromNS += "."
		logger.Debugf("picked %s record from namespace", recordFromNS)
		record, ok = recordsIdxByValue[recordFromNS]
		if !ok {
			continue
		}

		if record.Type != "A" {
			logger.Debugf("record %d has incorrect type %s: expected A", record.ID, record.Type)

			continue
		}

		record.Value = ip

		logger.Debugf("updating record %#v", record)

		err = c.updateDNSRecord(ctx, record)
		if err != nil {
			return fmt.Errorf("updating dns record: %w", err)
		}
	}

	return nil
}

func (c HTTPClient) updateDNSRecord(ctx context.Context, record dnsRecord) (err error) {
	const reqpath = "/v1/dns.record"

	type updateRequest struct {
		Value string `json:"value"`
	}

	idparam := strconv.FormatInt(record.ID, 10)

	_, err = c.performRequest(ctx, http.MethodPut, reqpath+"/"+idparam, updateRequest{
		Value: record.Value,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c HTTPClient) getDNSRecords(ctx context.Context, info dnsInfo) (records []dnsRecord, err error) {
	const reqpath = "/v1/dns.record"

	idparam := strconv.FormatInt(info.ID, 10)

	var response serviceResponse
	response, err = c.performRequest(ctx, http.MethodGet, reqpath+"/"+idparam, nil)
	if err != nil {
		return nil, err
	}

	err = response.unmarshalData(&records)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling data: %w", err)
	}

	return records, nil
}

func (c HTTPClient) getDNSInfos(ctx context.Context) (infos []dnsInfo, err error) {
	const reqpath = "/v1/dns"

	var response serviceResponse
	response, err = c.performRequest(ctx, http.MethodGet, reqpath, nil)
	if err != nil {
		return nil, err
	}

	err = response.unmarshalData(&infos)
	if err != nil {
		return nil, err
	}

	return infos, nil
}

func (c HTTPClient) authorize(ctx context.Context, user, pass string) (token string, err error) {
	const reqpath = "/v1/auth"

	type reqmodel struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type respmodel struct {
		Token string `json:"token"`
	}

	var response serviceResponse
	response, err = c.performRequest(ctx, http.MethodGet, reqpath, reqmodel{Email: user, Password: pass})
	if err != nil {
		return "", fmt.Errorf("performing request: %w", err)
	}

	var message respmodel
	err = response.unmarshalData(&message)
	if err != nil {
		return "", fmt.Errorf("unmarshalling data: %w", err)
	}

	return message.Token, nil
}

func (c HTTPClient) performRequest(ctx context.Context, method string, path string, data interface{}) (response serviceResponse, err error) {
	if !c.prepared {
		return response, model.ErrClientNotPrepared
	}

	logger := fcontext.Logger(ctx)
	requrl := c.url + path

	var reqbody io.Reader
	if data != nil {
		var reqdata []byte
		reqdata, err = json.Marshal(data)
		if err != nil {
			return response, fmt.Errorf("marshalling request data: %w", err)
		}

		reqbody = bytes.NewReader(reqdata)

		logger.Debugf("sending request to %s with data: %q", requrl, reqdata)
	} else {
		logger.Debugf("sending empty request to %s", requrl)
	}

	req, err := http.NewRequestWithContext(ctx, method, requrl, reqbody)
	if err != nil {
		return response, fmt.Errorf("making new request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", c.token)
	}

	req.Header.Set("Content-Type", "application/json")

	var resp *http.Response
	resp, err = c.client.Do(req)
	if err != nil {
		return response, fmt.Errorf("sending request: %w", err)
	}
	defer func() {
		errclose := resp.Body.Close()
		if errclose != nil {
			logger.Errf("closing response body: %v", err)
		}
	}()

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("reading body: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return response, fmt.Errorf("wrong credentials: %w", model.ErrUnauthorized)
	case http.StatusForbidden:
		return response, fmt.Errorf("missing permissions: %w", model.ErrForbidden)
	default:
		return response, fmt.Errorf("expected status code 200 got %d: %w", resp.StatusCode, model.ErrWrongStatusCode)
	}

	response, err = parseResponseData(body)
	if err != nil {
		return response, fmt.Errorf("parsing service response data: %w", err)
	}

	if !response.isOK() {
		return response, fmt.Errorf("request finished with error: %w", model.Error(response.StatusMsg))
	}

	return response, nil
}
