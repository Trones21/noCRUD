package nocrud

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Trones21/noCRUD/go/utils/apiclient"
	"github.com/Trones21/noCRUD/go/utils/dbclient"
	"github.com/Trones21/noCRUD/go/utils/fixtures"
	"github.com/Trones21/noCRUD/go/utils/perf"
)

// UserFixture is the fixture a flow logs in as by default. Point it somewhere
// else if your app's users live in another file.
var UserFixture = "users.json"

// Ctx is the single argument every flow receives — everything a flow needs
// that differs between one flow and the next.
//
// In Python all three of these are process-wide: print goes to stdout, the port
// comes from os.environ, and perf collection writes to a module global. Flows
// there are separate processes, so that works. Here they are goroutines, so the
// same three things are fields on this struct and a flow reaches them through
// c.Printf, c.NewClient and friends rather than through globals.
type Ctx struct {
	// FlowName is the name this flow was registered under.
	FlowName string
	// Env is the database and backend this flow owns.
	Env *Env

	ctx  context.Context
	out  io.Writer
	perf *perf.Collector

	// db is opened by Setup and closed by Close.
	db *dbclient.Client
}

// NewCtx builds a flow context. The runner calls this; a flow never has to.
//
// A nil ctx means context.Background and a nil out means os.Stdout, so a test
// can construct one with the two arguments it actually cares about.
func NewCtx(ctx context.Context, flowName string, env *Env, out io.Writer, collector *perf.Collector) *Ctx {
	if ctx == nil {
		ctx = context.Background()
	}
	if out == nil {
		out = os.Stdout
	}
	if env == nil {
		env = EnvFromProcess()
	}
	return &Ctx{
		FlowName: flowName,
		Env:      env,
		ctx:      ctx,
		out:      out,
		perf:     collector,
	}
}

// Context returns the run's context, cancelled when the runner is interrupted.
// Pass it to anything long-running a flow starts itself.
func (c *Ctx) Context() context.Context { return c.ctx }

// Out is where this flow's output goes.
//
// In parallel mode it is a buffer, not the terminal — the flow's log is printed
// in one piece when it finishes, so concurrent flows don't interleave into
// mush. Anything a flow prints should go here rather than to stdout.
func (c *Ctx) Out() io.Writer { return c.out }

// Printf writes to the flow's output.
func (c *Ctx) Printf(format string, args ...any) {
	fmt.Fprintf(c.out, format, args...)
}

// Close releases what the context opened. The runner defers it.
func (c *Ctx) Close() {
	if c.db != nil {
		if err := c.db.Close(c.ctx); err != nil {
			c.Printf("⚠️ Cleanup warning: closing database connection: %v\n", err)
		}
		c.db = nil
	}
}

// ============================================================================
//  Clients
// ============================================================================

// clientOptions points a new client at this flow's backend, output and perf
// collector.
func (c *Ctx) clientOptions() apiclient.Options {
	return apiclient.Options{
		BaseURL: c.Env.BaseURL(),
		Out:     c.out,
		Perf:    c.perf,
	}
}

// NewClient returns a client for this flow's backend, not logged in.
func (c *Ctx) NewClient() (*apiclient.Client, error) {
	return apiclient.New(c.clientOptions())
}

// NewFixtureUserClient returns a client logged in as a user from a fixture. The
// user has to exist already — Setup loads the fixtures that create them.
func (c *Ctx) NewFixtureUserClient(filename string, index int) (*apiclient.Client, error) {
	username, password, err := fixtures.Credentials(filename, index)
	if err != nil {
		return nil, err
	}
	return apiclient.NewWithUserViaFixture(c.clientOptions(), username, password)
}

// NewRandomUserClient registers a brand new user and returns a client logged in
// as them.
//
// This is how a multi-user flow gets its second actor: the rule under test
// usually only needs "somebody else", and registering one costs nothing and
// needs no fixture.
func (c *Ctx) NewRandomUserClient() (*apiclient.Client, error) {
	return apiclient.NewWithNewRandomUser(c.clientOptions())
}

// ============================================================================
//  Setup
// ============================================================================

// Setup is the first line of most flows: put the database in a known state and
// come back logged in.
//
//	api, err := c.Setup()
//	if err != nil {
//		return nil, err
//	}
//
// It empties the app's tables, reloads the minimal fixtures, and logs in as the
// first user in UserFixture — the Go counterpart of python/utils/common.py's
// setup().
//
// Clearing rather than re-migrating is deliberate: in parallel mode the
// database was migrated during provisioning, and doing it again per flow is the
// slowest thing the runner could do for no benefit.
func (c *Ctx) Setup(fixtureNames ...string) (*apiclient.Client, error) {
	db, err := c.DB()
	if err != nil {
		return nil, err
	}
	if err := db.ClearDataExceptMinimalFixtures(c.ctx, c.Env.Environ(), fixtureNames...); err != nil {
		return nil, fmt.Errorf("setup: %w", err)
	}
	return c.NewFixtureUserClient(UserFixture, 0)
}

// DB returns this flow's database connection, opening it on first use. Close
// takes care of it.
//
// Most flows never need this — they go through the API like a frontend would.
// It is here for the ones that have to assert on state the API doesn't expose,
// or set up a row the API can't create.
func (c *Ctx) DB() (*dbclient.Client, error) {
	if c.db != nil {
		return c.db, nil
	}
	db, err := dbclient.New(c.ctx, c.Env.DBConfig, c.out)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", c.Env.DBName, err)
	}
	c.db = db
	return db, nil
}
