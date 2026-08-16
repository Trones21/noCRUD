package flows

import (
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/crud"
)

func init() { nocrud.RegisterCRUD("actor", crudActorFlow) }

// crudActorFlow is the whole flow for an object with no dependencies: the
// generic create in utils/crud does the work, so there is nothing to write but
// the endpoint, the fixture and the field to update.
func crudActorFlow(c *nocrud.Ctx) (any, error) {
	api, err := c.Setup()
	if err != nil {
		return nil, err
	}

	return crud.Exec(c, api, "actor",
		crud.SimpleCreate("actor", "actors.json", 0, "id"),
		crud.UpdateDetails{Field: "first_name", NewValue: "Bill"},
	)
}
