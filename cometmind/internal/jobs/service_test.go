package jobs_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/cometline/cometmind/internal/db"
	"github.com/cometline/cometmind/internal/jobs"
	_ "modernc.org/sqlite"
)

func testJobsService(t *testing.T) *jobs.Service {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.EnsureSchema(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	return jobs.NewService(conn, nil, nil)
}

func TestCreateClaimComplete(t *testing.T) {
	svc := testJobsService(t)
	ctx := context.Background()

	job, err := svc.Create(ctx, jobs.CreateInput{
		Description:      "fix CI",
		DefinitionOfDone: "tests pass",
		Priority:         5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != jobs.StatusTodo {
		t.Fatalf("status=%s want todo", job.Status)
	}

	claimed, err := svc.Claim(ctx, job.ID, "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Status != jobs.StatusOngoing || claimed.AssignedSessionID != "sess-1" {
		t.Fatalf("claimed=%+v", claimed)
	}

	_, err = svc.Claim(ctx, job.ID, "sess-2")
	if err != jobs.ErrAlreadyClaimed {
		t.Fatalf("second claim err=%v want ErrAlreadyClaimed", err)
	}

	done, err := svc.Complete(ctx, job.ID, "sess-1", "all green")
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != jobs.StatusDone || done.Progress != "all green" {
		t.Fatalf("done=%+v", done)
	}
}

func TestReconcileOrphan(t *testing.T) {
	svc := testJobsService(t)
	ctx := context.Background()

	job, err := svc.Create(ctx, jobs.CreateInput{Description: "orphan test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Claim(ctx, job.ID, "sess-orphan"); err != nil {
		t.Fatal(err)
	}

	n, err := svc.Reconcile(ctx, func(sessionID string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("released=%d want 1", n)
	}
	got, err := svc.Get(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != jobs.StatusTodo {
		t.Fatalf("status=%s want todo", got.Status)
	}
}

func TestUpdateTodoOnlyInTodo(t *testing.T) {
	svc := testJobsService(t)
	ctx := context.Background()

	job, err := svc.Create(ctx, jobs.CreateInput{Description: "editable"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateTodo(ctx, job.ID, jobs.UpdateTodoInput{
		Description:      "updated",
		DefinitionOfDone: "done",
	}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Claim(ctx, job.ID, "sess-1"); err != nil {
		t.Fatal(err)
	}
	_, err = svc.UpdateTodo(ctx, job.ID, jobs.UpdateTodoInput{Description: "nope"}, "")
	if err != jobs.ErrNotEditable {
		t.Fatalf("err=%v want ErrNotEditable", err)
	}
}
