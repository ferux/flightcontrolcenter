package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ferux/flightcontrolcenter/internal/time"
)

// Application settings.
type Application struct {
	Debug          bool           `json:"debug"`
	HTTP           *HTTP          `json:"http"`
	GOBAPI         *GOB           `json:"gob_api"`
	SentryDSN      string         `json:"sentry_dsn"`
	NotifyTelegram NotifyTelegram `json:"notify_telegram"`
	ServerName     string         `json:"server_name"`
	DNSUpdater     DNSUpdater     `json:"dns_updater"`
}

type GOB struct {
	Listen string `json:"listen"`
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

// DNSUpdater is a service for updating dns records dynamically.
type DNSUpdater struct {
	Address string `json:"address"`
	// Namespaces is a collection of domain names that relates to specific
	// namespace. It needed to batch update dns records for multiple names
	// that belongs to single IP.
	Namespaces map[string][]string `json:"namespaces"`
}

// Parse parses config from file.
func Parse(path string) (Application, error) {
	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return Application{}, err
	}

	app := Application{}
	err = json.Unmarshal(fileBytes, &app)

	return app, err
}
