// Package crud runs the standard create/read/update/delete check against one
// endpoint — the Go counterpart of python/utils/crud.py.
//
// The point of a CRUD flow is uniformity: every endpoint gets the same four
// operations, so the summary is one line per endpoint and a schema change shows
// up as a column of ✘ rather than a wall of text.
package crud

import (
	"fmt"

	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/apiclient"
	"github.com/Trones21/noCRUD/go/utils/fixtures"
	"github.com/Trones21/noCRUD/go/utils/jsonx"
	"github.com/Trones21/noCRUD/go/utils/misc"
)

// Result is the outcome of the four operations. The runner prints it as
// "C:✔ R:✔ U:✔ D:✔".
type Result map[string]bool

// UpdateDetails says which field the update step should change, and to what.
type UpdateDetails struct {
	// Field is the field to modify.
	Field string
	// NewValue is what to set it to. Leave it nil to use a random string.
	NewValue any
	// Length is how long that random string should be. Zero means 12.
	Length int
}

// value resolves what the update step should write.
func (u UpdateDetails) value() any {
	if u.NewValue != nil {
		return u.NewValue
	}
	length := u.Length
	if length == 0 {
		length = 12
	}
	return misc.RandomString(length)
}

// CreateFunc creates the object under test and returns its id.
//
// It is a function rather than a fixture name because creating the object is
// the part that varies: a Trip may need a Driver, a Passenger and a Vehicle to
// exist first. Whatever that takes, it happens here, and the rest of the CRUD
// check doesn't have to know.
//
// This is also what makes create functions reusable as object builders. A
// CreateFunc that builds a whole dependency chain can be called from another
// flow — or from a seeding flow — instead of hand-managing ids.
//
// The id comes back as a jsonx.Value rather than a string so it can be used
// either way: Value.ID() for splicing into a URL, Value.Raw() for setting a
// foreign key on the next object without changing its JSON type.
type CreateFunc func(c *nocrud.Ctx, api *apiclient.Client) (jsonx.Value, error)

// SimpleCreate builds a CreateFunc for an object that stands on its own —
// nothing else has to exist first, and the fixture needs no massaging.
func SimpleCreate(endpoint, fixtureName string, index int, idField string) CreateFunc {
	return func(c *nocrud.Ctx, api *apiclient.Client) (jsonx.Value, error) {
		obj, err := fixtures.GetByIndex(fixtureName, index)
		if err != nil {
			return jsonx.Invalid(err), err
		}
		res, err := api.CreateObject(endpoint, obj)
		if err != nil {
			return res, err
		}
		return ID(res, endpoint, idField)
	}
}

// ID pulls the identifier out of a create response, with an error that says
// what actually came back when the field isn't there.
func ID(res jsonx.Value, endpoint, idField string) (jsonx.Value, error) {
	id := res.Get(idField)
	if id.Err() != nil {
		return id, fmt.Errorf("created %s but the response has no %q field: %v", endpoint, idField, res)
	}
	return id, nil
}

// Read fetches the object back by id.
func Read(api *apiclient.Client, endpoint, id string) (bool, error) {
	if _, err := api.GetObjectByID(endpoint, id, false); err != nil {
		return false, err
	}
	return true, nil
}

// Update reads the object, changes one field, writes it back, and confirms the
// change actually stuck — a 200 alone doesn't prove the backend kept the value.
func Update(c *nocrud.Ctx, api *apiclient.Client, endpoint, id string, details UpdateDetails) (bool, error) {
	obj, err := api.GetObjectByID(endpoint, id, true)
	if err != nil {
		return false, err
	}
	if obj.Map() == nil {
		return false, fmt.Errorf("update: GET %s/%s did not return an object: %v", endpoint, id, obj)
	}

	expected := details.value()
	obj.Set(details.Field, expected)

	res, err := api.UpdateObjectByID(endpoint, id, obj.Raw())
	if err != nil {
		return false, err
	}

	actual := res.Get(details.Field)
	if sameJSON(expected, actual) {
		return true, nil
	}
	c.Printf("Expected: %v  Actual: %v\n", expected, actual)
	return false, nil
}

// Delete removes the object.
func Delete(api *apiclient.Client, endpoint, id string) (bool, error) {
	if err := api.DeleteObjectByID(endpoint, id); err != nil {
		return false, err
	}
	return true, nil
}

// Exec runs create → read → update → delete against one endpoint.
//
// A step that errors stops the run and is returned along with the partial
// result, so the summary reports the actual failure rather than a cascade of
// follow-on ones (deleting an object that was never created tells you nothing).
func Exec(c *nocrud.Ctx, api *apiclient.Client, endpoint string, create CreateFunc, details UpdateDetails) (Result, error) {
	res := Result{}

	created, err := create(c, api)
	if err != nil {
		return res, err
	}
	// Getting this far means the create succeeded — anything else would have
	// come back as an error.
	res["create"] = true
	id := created.ID()

	if res["read"], err = Read(api, endpoint, id); err != nil {
		return res, err
	}
	if res["update"], err = Update(c, api, endpoint, id, details); err != nil {
		return res, err
	}
	if res["delete"], err = Delete(api, endpoint, id); err != nil {
		return res, err
	}
	return res, nil
}

// sameJSON compares what we sent with what came back, across the type shift a
// JSON round trip introduces: an int written as 5 comes back as float64(5), and
// a DRF CharField may return a number as "5".
func sameJSON(expected any, actual jsonx.Value) bool {
	if actual.Err() != nil {
		return false
	}
	return fmt.Sprint(jsonx.New(expected).ID()) == fmt.Sprint(actual.ID())
}
