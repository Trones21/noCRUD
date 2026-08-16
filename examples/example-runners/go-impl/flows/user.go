package flows

import (
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/apiclient"
	"github.com/Trones21/noCRUD/go/utils/crud"
	"github.com/Trones21/noCRUD/go/utils/fixtures"
	"github.com/Trones21/noCRUD/go/utils/jsonx"
	"github.com/Trones21/noCRUD/go/utils/misc"
)

func init() { nocrud.RegisterCRUD("users", crudUserFlow) }

func crudUserFlow(c *nocrud.Ctx) (any, error) {
	api, err := c.Setup()
	if err != nil {
		return nil, err
	}

	return crud.Exec(c, api, "users",
		createUser,
		crud.UpdateDetails{Field: "phone", NewValue: "5551234567"},
	)
}

// createUser creates a user from the fixture, under a name nobody has taken.
func createUser(c *nocrud.Ctx, api *apiclient.Client) (jsonx.Value, error) {
	user, err := fixtures.GetByIndex("users.json", 0)
	if err != nil {
		return jsonx.Invalid(err), err
	}
	// The users fixture has already been loaded by Setup, so reusing its
	// username would collide with the unique constraint.
	user["username"] = misc.RandomString(10)

	res, err := api.CreateObject("users", user)
	if err != nil {
		return res, err
	}
	return crud.ID(res, "users", "id")
}
