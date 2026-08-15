package crud_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Trones21/noCRUD/go/config"
	"github.com/Trones21/noCRUD/go/internal/testutil"
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/apiclient"
	"github.com/Trones21/noCRUD/go/utils/crud"
	"github.com/Trones21/noCRUD/go/utils/jsonx"
)

// writeFixtures points the fixtures package at a temporary directory holding
// one universes.json, so the CRUD path can be exercised end to end.
func writeFixtures(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	body := `[{"model": "api.universe", "pk": 1, "fields": {"name": "Breaking Bad Universe", "description": "Includes Better Call Saul"}}]`
	if err := os.WriteFile(filepath.Join(dir, "universes.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(config.EnvFixturesPath, dir)
}

func setup(t *testing.T) (*nocrud.Ctx, *apiclient.Client, *testutil.DRF) {
	t.Helper()
	writeFixtures(t)

	drf := testutil.NewDRF(t)
	var out bytes.Buffer
	c := nocrud.NewCtx(nil, "test", drf.Env(), &out, nil)

	client, err := c.NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := client.Login("user", "pass"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	return c, client, drf
}

func TestExecReportsAllFourOperations(t *testing.T) {
	c, client, drf := setup(t)

	res, err := crud.Exec(c, client, "universe",
		crud.SimpleCreate("universe", "universes.json", 0, "id"),
		crud.UpdateDetails{Field: "name", NewValue: "A whole new world"})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}

	for _, op := range []string{"create", "read", "update", "delete"} {
		if !res[op] {
			t.Errorf("%s = false, want true (result: %v)", op, res)
		}
	}
	if left := drf.Objects("universe"); len(left) != 0 {
		t.Errorf("delete left %d objects behind", len(left))
	}
}

// A 200 on the update is not the same as the value having changed. Exec has to
// read the response back, or a backend that silently drops the field passes.
func TestExecFlagsAnUpdateThatDidNotStick(t *testing.T) {
	c, client, drf := setup(t)
	drf.OnUpdate = func(resource string, id int, obj map[string]any) (map[string]any, bool) {
		return map[string]any{"id": float64(id), "name": "not what you asked for"}, true
	}

	res, err := crud.Exec(c, client, "universe",
		crud.SimpleCreate("universe", "universes.json", 0, "id"),
		crud.UpdateDetails{Field: "name", NewValue: "A whole new world"})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res["update"] {
		t.Error("update = true, want false when the backend returned a different value")
	}
	if !res["delete"] {
		t.Error("a failed comparison should not stop the remaining operations")
	}
}

func TestExecStopsAtTheFirstHardFailure(t *testing.T) {
	c, client, drf := setup(t)
	drf.OnCreate = func(resource string, obj map[string]any) (int, map[string]any, bool) {
		return 400, map[string]any{"name": []any{"This field is required."}}, true
	}

	res, err := crud.Exec(c, client, "universe",
		crud.SimpleCreate("universe", "universes.json", 0, "id"),
		crud.UpdateDetails{Field: "name", NewValue: "x"})
	if err == nil {
		t.Fatal("expected the create failure to be returned")
	}
	if len(res) != 0 {
		t.Errorf("result = %v, want nothing recorded when the create failed", res)
	}
	if got := apiclient.StatusOf(err); got != 400 {
		t.Errorf("status = %d, want 400", got)
	}
}

func TestUpdateDetailsGeneratesAValueWhenNoneIsGiven(t *testing.T) {
	c, client, _ := setup(t)

	res, err := crud.Exec(c, client, "universe",
		crud.SimpleCreate("universe", "universes.json", 0, "id"),
		crud.UpdateDetails{Field: "name", Length: 49})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if !res["update"] {
		t.Error("update = false, want true with a generated value")
	}
}

// Numbers survive the JSON round trip as float64; the comparison has to see
// through that, or every numeric field looks like a failed update.
func TestUpdateComparesAcrossJSONNumberTypes(t *testing.T) {
	c, client, _ := setup(t)

	res, err := crud.Exec(c, client, "universe",
		crud.SimpleCreate("universe", "universes.json", 0, "id"),
		crud.UpdateDetails{Field: "rank", NewValue: -1})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if !res["update"] {
		t.Error("update = false, want true for an integer value")
	}
}

func TestSimpleCreateReportsAMissingIDField(t *testing.T) {
	c, client, drf := setup(t)
	drf.OnCreate = func(resource string, obj map[string]any) (int, map[string]any, bool) {
		return 201, map[string]any{"name": "no id here"}, true
	}

	_, err := crud.SimpleCreate("universe", "universes.json", 0, "id")(c, client)
	if err == nil {
		t.Fatal("expected an error when the response has no id")
	}
}

func TestIDAcceptsStringAndNumericIdentifiers(t *testing.T) {
	numeric := jsonx.New(map[string]any{"id": float64(42)})
	id, err := crud.ID(numeric, "universe", "id")
	if err != nil {
		t.Fatalf("ID: %v", err)
	}
	if id.ID() != "42" {
		t.Errorf("numeric id = %q, want 42", id.ID())
	}

	uuid := jsonx.New(map[string]any{"id": "0b8e-uuid"})
	id, err = crud.ID(uuid, "universe", "id")
	if err != nil {
		t.Fatalf("ID: %v", err)
	}
	if id.ID() != "0b8e-uuid" {
		t.Errorf("string id = %q", id.ID())
	}
}
