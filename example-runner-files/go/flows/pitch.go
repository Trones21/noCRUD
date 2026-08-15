package flows

import (
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/apiclient"
	"github.com/Trones21/noCRUD/go/utils/crud"
	"github.com/Trones21/noCRUD/go/utils/fixtures"
	"github.com/Trones21/noCRUD/go/utils/jsonx"
)

func init() { nocrud.RegisterCRUD("pitch", crudPitchFlow) }

func crudPitchFlow(c *nocrud.Ctx) (any, error) {
	api, err := c.Setup()
	if err != nil {
		return nil, err
	}

	return crud.Exec(c, api, "pitch",
		createPitch,
		crud.UpdateDetails{Field: "pitch_text", NewValue: "updated_text"},
	)
}

// createPitch creates the pitch, and the four-object chain it hangs off:
// universe → production → character → pitch. This is the case that makes
// object builders worth having — none of it involves a hardcoded id.
func createPitch(c *nocrud.Ctx, api *apiclient.Client) (jsonx.Value, error) {
	characterID, err := createCharacter(c, api)
	if err != nil {
		return characterID, err
	}

	obj, err := fixtures.GetByIndex("backstory_pitches.json", 0)
	if err != nil {
		return jsonx.Invalid(err), err
	}
	obj["character"] = characterID.Raw()

	res, err := api.CreateObject("pitch", obj)
	if err != nil {
		return res, err
	}
	return crud.ID(res, "pitch", "id")
}
