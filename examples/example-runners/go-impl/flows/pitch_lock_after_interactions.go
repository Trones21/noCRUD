package flows

import (
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/fixtures"
)

func init() {
	nocrud.Register(nocrud.Flow{
		Name: "pitch_lock_after_interactions",
		Kind: nocrud.Request,
		Doc:  "A pitch cannot be edited once someone has commented on it",
		Fn:   pitchLockAfterInteractionsFlow,
	})
}

// pitchLockAfterInteractionsFlow is the kind of thing CRUD checks can't reach:
// a rule that only exists in the interaction between two users.
//
//	author  → creates a pitch
//	critic  → comments on it        (this is what locks it)
//	author  → tries to edit it      (must now fail)
//
// It is also executable documentation. The rule lives in three places in the
// backend — a save() override, a lock_if_interacted() helper and the comment
// viewset that calls it — and this is the only place you can read it in one go.
func pitchLockAfterInteractionsFlow(c *nocrud.Ctx) (any, error) {
	author, err := c.Setup()
	if err != nil {
		return nil, err
	}

	// Build the pitch (and the universe → production → character chain under
	// it) with the same object builder the CRUD flow uses.
	pitchID, err := createPitch(c, author)
	if err != nil {
		return nil, err
	}

	// A second user, with their own session. Nothing is shared between the two
	// clients but the backend.
	critic, err := c.NewRandomUserClient()
	if err != nil {
		return nil, err
	}

	comment, err := fixtures.GetByIndex("pitch_comments.json", 0)
	if err != nil {
		return nil, err
	}
	comment["pitch"] = pitchID.Raw()
	if _, err := critic.CreateObject("pitch_comment", comment); err != nil {
		return nil, err
	}

	// The pitch is locked now. The author owns it and is authenticated, and it
	// still has to be refused.
	//
	// PATCH rather than PUT, and the difference matters more than it looks.
	// PUT replaces the object, so a body of just pitch_text is rejected at
	// validation for the fields it left out — the edit fails, but on a
	// technicality, before the lock is ever consulted. The flow would pass
	// against a backend with no locking at all. PATCH sends only the field
	// being changed, so the refusal is the rule.
	//
	// Note this asserts *that* the edit fails, not which status it fails with:
	// the backend raises a Django ValidationError from save(), which DRF
	// doesn't translate, so it surfaces as a 500 rather than a 400. Worth
	// knowing about — and exactly the sort of thing running the flow tells you.
	err = nocrud.ExpectFail(c, "editing a locked pitch", func() error {
		_, err := author.PatchObjectByID("pitch", pitchID.ID(), map[string]any{
			"pitch_text": "trying to sneak in an edit",
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	return "pitch locked after interaction, as expected", nil
}
