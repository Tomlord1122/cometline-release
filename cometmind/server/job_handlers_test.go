package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cometline/cometmind/internal/config"
	"github.com/cometline/cometmind/internal/jobs"
	"github.com/cometline/cometmind/internal/session"
	"github.com/cometline/cometmind/internal/store"
)

func TestJobHandlersCreateListClaim(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := store.OpenSQLite(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	sessions := session.New(sqlDB)
	jobSvc := jobs.NewService(sqlDB, nil, nil)
	engine, err := New(Deps{
		Config:    config.Defaults(),
		Sessions:  sessions,
		Jobs:      jobSvc,
		NewRunner: func(session.Session, string) (Runner, error) { return nil, nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	createBody := `{"description":"fix tests","definition_of_done":"green","priority":3}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/jobs?ready_only=true", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status=%d", w.Code)
	}

	claimBody := `{"session_id":"sess-1"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/jobs/"+created.ID+"/claim", bytes.NewBufferString(claimBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("claim status=%d body=%s", w.Code, w.Body.String())
	}
}
