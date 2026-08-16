package nocrud

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Trones21/noCRUD/go/utils/dbclient"
)

// Env is one flow's world: the database it owns, the port its backend listens
// on, and the handles needed to take both down again.
//
// In parallel mode every flow has its own, which is the whole reason this type
// exists. The Python runner can keep the equivalent in os.environ because each
// flow is a separate process; here flows are goroutines sharing one process, so
// a per-flow value has to be passed rather than set globally. Environ is where
// it converts back into real environment variables, for the subprocesses that
// do still read them.
type Env struct {
	// DBName is the database provisioned for this flow.
	DBName string
	// AppPort is the port this flow's backend listens on.
	AppPort int
	// DBConfig connects to DBName as the app's user.
	DBConfig dbclient.Config

	// Admin is the superuser connection that created the database and will
	// drop it. Nil when the flow didn't provision one (serial mode).
	Admin *dbclient.Client
	// Proc is the backend process, when the runner started it.
	Proc *exec.Cmd

	// PersistDB keeps the database after the run. The runner sets it on
	// failure, so the state that broke a flow is still there to look at.
	PersistDB bool
}

// DefaultAppPort is where the backend is assumed to be in serial mode, matching
// Django's own default.
const DefaultAppPort = 8000

// EnvFromProcess reads the environment the process was started with — serial
// mode, where you started the app yourself and the runner just talks to it.
func EnvFromProcess() *Env {
	cfg := dbclient.ConfigFromEnv()

	port := DefaultAppPort
	if raw := os.Getenv("APP_PORT"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			port = parsed
		}
	}

	return &Env{
		DBName:   cfg.DBName,
		AppPort:  port,
		DBConfig: cfg,
	}
}

// BaseURL is the API root of this flow's backend.
//
// 127.0.0.1 rather than localhost: on a host where localhost resolves to ::1
// first, a backend bound only to IPv4 is unreachable by name, and the failure
// reads as "is it running?" when it is.
func (e *Env) BaseURL() string {
	port := DefaultAppPort
	if e != nil && e.AppPort != 0 {
		port = e.AppPort
	}
	return fmt.Sprintf("http://127.0.0.1:%d/api", port)
}

// Environ renders the environment for a subprocess — the app, manage.py, psql.
//
// It starts from the real environment so everything unrelated (DJANGO_KEY, PATH,
// a virtualenv) carries through, then overrides the per-flow values. The
// override is what points the app at *this* flow's database: settings.py reads
// DB_NAME, and provisioning.DBMatchCheck fails the flow early if it somehow
// resolves to anything else.
func (e *Env) Environ() []string {
	overrides := map[string]string{
		"DB_NAME": e.DBName,
		"DB_USER": e.DBConfig.User,
		"DB_PASS": e.DBConfig.Pass,
		"DB_HOST": e.DBConfig.Host,
		"DB_PORT": e.DBConfig.Port,
	}
	if e.AppPort != 0 {
		overrides["APP_PORT"] = strconv.Itoa(e.AppPort)
	}
	// PGPASSWORD covers the psql and createdb paths, which don't read DB_PASS.
	if e.DBConfig.Pass != "" {
		overrides["PGPASSWORD"] = e.DBConfig.Pass
	}

	out := make([]string, 0, len(os.Environ())+len(overrides))
	for _, kv := range os.Environ() {
		key, _, found := strings.Cut(kv, "=")
		if found {
			if _, replaced := overrides[key]; replaced {
				continue
			}
		}
		out = append(out, kv)
	}
	for key, value := range overrides {
		if value == "" {
			continue
		}
		out = append(out, key+"="+value)
	}
	return out
}
