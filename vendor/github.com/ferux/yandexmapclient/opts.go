package yandexmapclient

import "time"

// ClientOption applies options to client
type ClientOption func(*Client) error

func WithLogger(l Logger) ClientOption {
	return func(c *Client) error {
		if l == nil {
			l = &nopLogger{}
		}
		c.logger = l
		return nil
	}
}

func WithTimeout(t time.Duration) ClientOption {
	return func(c *Client) error {
		c.client.Timeout = t
		return nil
	}
}

func WithCsrfToken(token string) ClientOption {
	return func(c *Client) error {
		c.csrfToken = token
		return nil
	}
}

func WithHost(host string) ClientOption {
	return func(c *Client) error {
		c.host = host
		return nil
	}
}

func WithLocale(locale string) ClientOption {
	return func(c *Client) error {
		c.locale = locale
		return nil
	}
}
