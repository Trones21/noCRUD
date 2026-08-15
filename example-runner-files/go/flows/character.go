package flows

import (
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/apiclient"
	"github.com/Trones21/noCRUD/go/utils/crud"
	"github.com/Trones21/noCRUD/go/utils/fixtures"
	"github.com/Trones21/noCRUD/go/utils/jsonx"
)

func init() { nocrud.RegisterCRUD("character", crudCharacterFlow) }

func crudCharacterFlow(c *nocrud.Ctx) (any, error) {
	api, err := c.Setup()
	if err != nil {
		return nil, err
	}

	return crud.Exec(c, api, "character",
		createCharacter,
		crud.UpdateDetails{Field: "name", NewValue: "WW"},
	)
}

// createCharacter creates the character, and the production (and universe) it
// appears in.
func createCharacter(c *nocrud.Ctx, api *apiclient.Client) (jsonx.Value, error) {
	productionID, err := createProduction(c, api)
	if err != nil {
		return productionID, err
	}

	obj, err := fixtures.GetByIndex("characters.json", 0)
	if err != nil {
		return jsonx.Invalid(err), err
	}
	obj["productions"] = []any{productionID.Raw()}

	res, err := api.CreateObject("character", obj)
	if err != nil {
		return res, err
	}
	return crud.ID(res, "character", "id")
}
