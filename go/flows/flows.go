// Package flows is where your flows live.
//
// A flow registers itself from its own init(), so adding one is a matter of
// dropping a file in this directory — nothing else in the runner changes:
//
//	package flows
//
//	func init() { nocrud.RegisterCRUD("actor", crudActorFlow) }
//
//	func crudActorFlow(c *nocrud.Ctx) (any, error) {
//		api, err := c.Setup()
//		if err != nil {
//			return nil, err
//		}
//		return crud.Exec(c, api, "actor",
//			crud.SimpleCreate("actor", "actors.json", 0, "id"),
//			crud.UpdateDetails{Field: "first_name", NewValue: "Bill"})
//	}
//
// The package ships empty on purpose, the same way python/flows/ does: these
// are your tests, not noCRUD's. Ready-made examples that run against the
// bundled example_app are in example-runner-files/go/flows/ — copy them in:
//
//	cp example-runner-files/go/flows/*.go go/flows/
//
// Then check they registered before you rely on them:
//
//	go run ./cmd/nocrud -l
package flows
