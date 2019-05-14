package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ferux/flightcontrolcenter/internal/time"
)

// Application settings
type Application struct {
	Debug bool       `json:"debug,omitempty"`
	HTTP  configHTTP `json:"http,omitempty"`
}

type configHTTP struct {
	Listen  string        `json:"listen,omitempty"`
	Timeout time.Duration `json:"timeout,omitempty"`
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
