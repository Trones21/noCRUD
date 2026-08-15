package runners_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Trones21/noCRUD/go/config"
	"github.com/Trones21/noCRUD/go/internal/testutil"
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/runners"
	"github.com/Trones21/noCRUD/go/utils/apiclient"
	"github.com/Trones21/noCRUD/go/utils/crud"
	"github.com/Trones21/noCRUD/go/utils/jsonx"
	"github.com/Trones21/noCRUD/go/utils/perf"
)

// This is the whole runner end to end — registry, runner, context, API client,
// CRUD helpers, perf collection and the summary — against a fake DRF backend.
// Everything but provisioning, which needs a real Django app and database.

func fixtureDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"universes.json":   `[{"model": "api.universe", "pk": 1, "fields": {"name": "Breaking Bad Universe"}}]`,
		"productions.json": `[{"model": "api.production", "pk": 1, "fields": {"title": "Breaking Bad"}}]`,
		"actors.json":      `[{"model": "api.actor", "pk": 1, "fields": {"first_name": "Bryan", "last_name": "Cranston"}}]`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(config.EnvFixturesPath, dir)
}

// login stands in for Ctx.Setup, which resets a database this test doesn't have.
func login(c *nocrud.Ctx) (*apiclient.Client, error) {
	client, err := c.NewClient()
	if err != nil {
		return nil, err
	}
	return client, client.Login("var_undecided", "fixture_pass")
}

func TestEndToEndCRUDAndBusinessLogicFlows(t *testing.T) {
	fixtureDir(t)
	drf := testutil.NewDRF(t)

	// A CRUD flow, exactly as a flow file would write it.
	actor := nocrud.Flow{Name: "actor", Kind: nocrud.CRUD, Fn: func(c *nocrud.Ctx) (any, error) {
		api, err := login(c)
		if err != nil {
			return nil, err
		}
		return crud.Exec(c, api, "actor",
			crud.SimpleCreate("actor", "actors.json", 0, "id"),
			crud.UpdateDetails{Field: "first_name", NewValue: "Bill"})
	}}

	// A dependency chain: production needs a universe, and the id is threaded
	// through from the response rather than hardcoded.
	production := nocrud.Flow{Name: "production", Kind: nocrud.CRUD, Fn: func(c *nocrud.Ctx) (any, error) {
		api, err := login(c)
		if err != nil {
			return nil, err
		}
		createProduction := func(c *nocrud.Ctx, api *apiclient.Client) (jsonx.Value, error) {
			universe, err := crud.SimpleCreate("universe", "universes.json", 0, "id")(c, api)
			if err != nil {
				return universe, err
			}
			obj := map[string]any{"title": "Breaking Bad", "universe": universe.Raw()}
			res, err := api.CreateObject("production", obj)
			if err != nil {
				return res, err
			}
			return crud.ID(res, "production", "id")
		}
		return crud.Exec(c, api, "production", createProduction,
			crud.UpdateDetails{Field: "title", NewValue: "Reconstructing Goodman"})
	}}

	// A multi-user rule: the second user may not touch the first user's object.
	permissions := nocrud.Flow{Name: "permissions", Kind: nocrud.Request, Fn: func(c *nocrud.Ctx) (any, error) {
		author, err := login(c)
		if err != nil {
			return nil, err
		}
		created := nocrud.Must(author.CreateObject("universe", map[string]any{"name": "private"}))

		other, err := c.NewRandomUserClient()
		if err != nil {
			return nil, err
		}
		// The fake allows this; what's under test is that ExpectStatus reports
		// the mismatch rather than passing silently.
		err = nocrud.ExpectStatus(c, "another user editing it", 403, func() error {
			_, err := other.UpdateObjectByID("universe", created.Get("id").ID(), map[string]any{"name": "hijacked"})
			return err
		})
		if err == nil {
			return nil, nil
		}
		return "expected-failure-detected", nil
	}}

	var out bytes.Buffer
	results := runners.Run(context.Background(), []nocrud.Flow{actor, production, permissions}, runners.Options{
		Parallel: true,
		Out:      &out,
		Provision: func(ctx context.Context, flowName string, w io.Writer) (*nocrud.Env, error) {
			return drf.Env(), nil
		},
		Cleanup: func(ctx context.Context, env *nocrud.Env, w io.Writer) {},
	})

	if got := results[0].Formatted; !strings.Contains(got, "C:") || strings.Contains(got, "✘") {
		t.Errorf("actor = %q, want four ticks", got)
	}
	if got := results[1].Formatted; strings.Contains(got, "✘") {
		t.Errorf("production = %q, want four ticks", got)
	}
	if results[2].Err != nil {
		t.Errorf("permissions flow errored: %v", results[2].Err)
	}
	if got := results[2].Formatted; got != "expected-failure-detected" {
		t.Errorf("permissions = %q — the unexpected success should have been caught", got)
	}

	if !runners.PrintSummary(&out, results) {
		t.Errorf("summary reported a failure:\n%s", out.String())
	}
}

func TestEndToEndCollectsTimings(t *testing.T) {
	fixtureDir(t)
	drf := testutil.NewDRF(t)

	dir := t.TempDir()
	t.Setenv(perf.EnvEnabled, "1")
	t.Setenv(perf.EnvRunID, "")
	t.Setenv(perf.EnvSHA, "testsha")
	t.Setenv(perf.EnvDir, filepath.Join(dir, "perf"))

	run, err := perf.InitRun(dir)
	if err != nil {
		t.Fatalf("InitRun: %v", err)
	}

	flow := nocrud.Flow{Name: "actor", Kind: nocrud.CRUD, Fn: func(c *nocrud.Ctx) (any, error) {
		api, err := login(c)
		if err != nil {
			return nil, err
		}
		return crud.Exec(c, api, "actor",
			crud.SimpleCreate("actor", "actors.json", 0, "id"),
			crud.UpdateDetails{Field: "first_name", NewValue: "Bill"})
	}}

	runners.Run(context.Background(), []nocrud.Flow{flow}, runners.Options{
		Parallel: true,
		Run:      run,
		Out:      io.Discard,
		Provision: func(ctx context.Context, flowName string, w io.Writer) (*nocrud.Env, error) {
			return drf.Env(), nil
		},
		Cleanup: func(ctx context.Context, env *nocrud.Env, w io.Writer) {},
	})

	records, err := perf.LoadRecords(run.Dir)
	if err != nil {
		t.Fatalf("LoadRecords: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("no timings were persisted")
	}

	// The op/endpoint pair is what makes a baseline comparable across runs —
	// and across the two implementations, which use the same op names.
	want := map[string]bool{
		"create_object": false, "get_object_by_id": false,
		"update_object_by_id": false, "delete_object_by_id": false,
	}
	for _, rec := range records {
		if rec.Flow != "actor" {
			t.Errorf("record tagged with flow %q", rec.Flow)
		}
		if rec.Endpoint == nil || *rec.Endpoint != "actor" {
			t.Errorf("record has endpoint %v, want actor", rec.Endpoint)
		}
		want[rec.Op] = true
	}
	for op, seen := range want {
		if !seen {
			t.Errorf("no timing recorded for %s", op)
		}
	}
}
