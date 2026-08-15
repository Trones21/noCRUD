package flows

import (
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/apiclient"
	"github.com/Trones21/noCRUD/go/utils/crud"
	"github.com/Trones21/noCRUD/go/utils/fixtures"
	"github.com/Trones21/noCRUD/go/utils/jsonx"
)

func init() { nocrud.RegisterCRUD("vote", crudVoteFlow) }

func crudVoteFlow(c *nocrud.Ctx) (any, error) {
	api, err := c.Setup()
	if err != nil {
		return nil, err
	}

	return crud.Exec(c, api, "vote",
		createVote,
		// A number rather than a string — UpdateDetails takes any JSON value.
		crud.UpdateDetails{Field: "value", NewValue: -1},
	)
}

// createVote creates the vote, and the whole chain the vote needs:
// universe → production → character → pitch → vote.
func createVote(c *nocrud.Ctx, api *apiclient.Client) (jsonx.Value, error) {
	pitchID, err := createPitch(c, api)
	if err != nil {
		return pitchID, err
	}

	obj, err := fixtures.GetByIndex("votes.json", 0)
	if err != nil {
		return jsonx.Invalid(err), err
	}
	obj["pitch"] = pitchID.Raw()

	res, err := api.CreateObject("vote", obj)
	if err != nil {
		return res, err
	}
	return crud.ID(res, "vote", "id")
}
