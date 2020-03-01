package fcchttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ferux/flightcontrolcenter/internal/model"
	"github.com/ferux/flightcontrolcenter/internal/templates"

	"github.com/matryer/is"
)

func TestGetInfo(t *testing.T) {
	var is = is.New(t)
	var now = time.Now()

	var appInfo = model.ApplicationInfo{
		Branch:   "master",
		Revision: "revision",
	}

	r := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	w := httptest.NewRecorder()
	api := &HTTP{
		bootTime:     now,
		requestCount: 10,
	}

	api.handleInfo(appInfo)(w, r)
	exp := templates.MarshalData{
		Revision: "revision",
		Branch:   "master",
		BootTime: now.String(),
		// because when marshaling we cut everything after the point
		Uptime:       float64(int(time.Since(now).Seconds())),
		RequestCount: int(api.requestCount),
	}

	is.Equal(w.Code, http.StatusOK)

	var got templates.MarshalData
	_ = json.Unmarshal(w.Body.Bytes(), &got)

	is.Equal(exp, got)
}
