package flows

import (
	"fmt"

	"github.com/Trones21/noCRUD/go/nocrud"
)

func init() {
	nocrud.Register(nocrud.Flow{
		Name: "seed_pitches",
		Kind: nocrud.Request,
		Doc:  "Fill the database with valid pitch object graphs, via the API",
		Fn:   seedPitchesFlow,
	})
}

// seedPitchesFlow shows the other use for object builders: not testing, but
// filling a database.
//
// It beats loading fixtures directly on two counts. Every object goes in
// through the actual API, so nothing lands in a state the endpoints wouldn't
// produce; and the ids come from the responses, so the relationships are right
// without anyone hand-managing primary keys.
//
// Run it against a database you want to keep:
//
//	go run ./cmd/nocrud -f seed_pitches --serial
func seedPitchesFlow(c *nocrud.Ctx) (any, error) {
	const count = 5

	api, err := c.Setup()
	if err != nil {
		return nil, err
	}

	for i := 0; i < count; i++ {
		if _, err := createPitch(c, api); err != nil {
			return nil, fmt.Errorf("seeding pitch %d of %d: %w", i+1, count, err)
		}
	}

	return fmt.Sprintf("seeded %d pitches (with their characters, productions and universes)", count), nil
}
