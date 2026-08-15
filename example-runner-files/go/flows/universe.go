package flows

import (
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/apiclient"
	"github.com/Trones21/noCRUD/go/utils/crud"
	"github.com/Trones21/noCRUD/go/utils/fixtures"
	"github.com/Trones21/noCRUD/go/utils/jsonx"
)

// Registering from init() is the Go equivalent of the Python runner's folder
// collector: dropping this file into the flows package is the whole of it.
func init() { nocrud.RegisterCRUD("universe", crudUniverseFlow) }

func crudUniverseFlow(c *nocrud.Ctx) (any, error) {
	api, err := c.Setup()
	if err != nil {
		return nil, err
	}

	return crud.Exec(c, api, "universe",
		createUniverse,
		crud.UpdateDetails{Field: "name", NewValue: "A whole new world"},
	)
}

// createUniverse creates the universe.
//
// It is a plain function of the CreateFunc shape, so the flows further down the
// dependency chain (production → character → pitch → vote) can call it instead
// of inventing their own universe.
func createUniverse(c *nocrud.Ctx, api *apiclient.Client) (jsonx.Value, error) {
	obj, err := fixtures.GetByIndex("universes.json", 0)
	if err != nil {
		return jsonx.Invalid(err), err
	}
	res, err := api.CreateObject("universe", obj)
	if err != nil {
		return res, err
	}
	return crud.ID(res, "universe", "id")
}
