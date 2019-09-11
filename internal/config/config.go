package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ferux/flightcontrolcenter/internal/time"
)

// Application settings
type Application struct {
	Debug          bool                 `json:"debug,omitempty"`
	HTTP           configHTTP           `json:"http,omitempty"`
	SentryDSN      string               `json:"sentry_dsn,omitempty"`
	NotifyTelegram configNotifyTelegram `json:"notify_telegram"`
}

type configHTTP struct {
	Listen  string        `json:"listen,omitempty"`
	Timeout time.Duration `json:"timeout,omitempty"`
}

type configNotifyTelegram struct {
	API    string `json:"api"`
	ChatID string `json:"chat_id"`
}

// Parse parses config from file
func Parse(path string) (Application, error) {
	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return Application{}, err
	}

	var app = Application{}
	err = json.Unmarshal(fileBytes, &app)

	return app, err
}
