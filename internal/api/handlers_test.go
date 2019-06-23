package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	fcc "github.com/ferux/flightcontrolcenter"
	"github.com/ferux/flightcontrolcenter/internal/templates"
)

func BenchmarkGetInfo(b *testing.B) {
	b.ReportAllocs()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	rec := httptest.NewRecorder()
	api := &HTTP{}

	for i := 0; i < b.N; i++ {
		api.handleInfo(rec, request)
		if rec.Code != http.StatusOK {
			b.Fail()
		}
		rec.Flush()
	}
}

func TestGetInfo(t *testing.T) {
	fcc.Branch = "master"
	fcc.Env = "production"
	fcc.Revision = "00000000"
	now := time.Now()

	r := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	w := httptest.NewRecorder()
	api := &HTTP{
		bootTime:     now,
		requestCount: 10,
	}

	api.handleInfo(w, r)
	exp := templates.MarshalData{
		Revision:     fcc.Revision,
		Branch:       fcc.Branch,
		BootTime:     now.String(),
		// because when marshaling we cut everything after the point
		Uptime:       float64(int(time.Since(now).Seconds())),
		RequestCount: int(api.requestCount),
	}

	if w.Code != http.StatusOK {
		t.Fatalf("exp %d got %d", http.StatusOK, w.Code)
	}

	var got templates.MarshalData
	_ = json.Unmarshal(w.Body.Bytes(), &got)

	if exp != got {
		t.Fatalf("exp %#v got %#v", exp, got)
	}
}
