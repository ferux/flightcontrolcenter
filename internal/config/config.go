package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ferux/flightcontrolcenter/internal/time"
)

// Application settings
type Application struct {
	Debug          bool           `json:"debug"`
	HTTP           HTTP           `json:"http"`
	SentryDSN      string         `json:"sentry_dsn"`
	NotifyTelegram NotifyTelegram `json:"notify_telegram"`
}

type HTTP struct {
	Listen  string        `json:"listen"`
	Timeout time.Duration `json:"timeout"`
}

type NotifyTelegram struct {
	API    string `json:"api"`
	ChatID string `json:"chat_id"`
}

type Keeper struct {
	Listen   string `json:"listen"`
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
	Name     string `json:"name"`
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
