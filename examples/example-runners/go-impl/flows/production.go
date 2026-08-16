package flows

import (
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/apiclient"
	"github.com/Trones21/noCRUD/go/utils/crud"
	"github.com/Trones21/noCRUD/go/utils/fixtures"
	"github.com/Trones21/noCRUD/go/utils/jsonx"
)

func init() { nocrud.RegisterCRUD("production", crudProductionFlow) }

func crudProductionFlow(c *nocrud.Ctx) (any, error) {
	api, err := c.Setup()
	if err != nil {
		return nil, err
	}

	return crud.Exec(c, api, "production",
		createProduction,
		crud.UpdateDetails{Field: "title", NewValue: "Reconstructing Goodman"},
	)
}

// createProduction creates the production, and the universe it belongs to.
func createProduction(c *nocrud.Ctx, api *apiclient.Client) (jsonx.Value, error) {
	// Not strictly necessary in this example — productions could point at an
	// existing universe — but this is the technique: build the dependency, then
	// the thing that needs it.
	universeID, err := createUniverse(c, api)
	if err != nil {
		return universeID, err
	}

	obj, err := fixtures.GetByIndex("productions.json", 0)
	if err != nil {
		return jsonx.Invalid(err), err
	}
	obj["universe"] = universeID.Raw()

	res, err := api.CreateObject("production", obj)
	if err != nil {
		return res, err
	}
	return crud.ID(res, "production", "id")
}
