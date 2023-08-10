package register

import (
	"errors"
	"fmt"
	"xk6-environment/pkg/environment"
	"xk6-environment/pkg/fs"
	"xk6-environment/pkg/kubernetes"

	"github.com/dop251/goja"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

// init is called by the Go runtime at application startup.
func init() {
	modules.Register("k6/x/environment", New())
}

type (
	// RootModule is the global module instance that will create module
	// instances for each VU.
	RootModule struct{}

	// ModuleInstance represents an instance of the JS module.
	ModuleInstance struct {
		// vu provides methods for accessing internal k6 objects for a VU
		vu modules.VU
		// comparator is the exported type
		e *environment.Environment
	}
)

// Ensure the interfaces are implemented correctly.
var (
	_ modules.Instance = &ModuleInstance{}
	_ modules.Module   = &RootModule{}
)

// New returns a pointer to a new RootModule instance.
func New() *RootModule {
	return &RootModule{}
}

// NewModuleInstance implements the modules.Module interface returning a new instance for each VU.
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &ModuleInstance{
		vu: vu,
		e:  &environment.Environment{VU: vu}, //, nil, nil, ""},
	}
}

// Exports implements the modules.Instance interface and returns the exported types for the JS module.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{Named: map[string]interface{}{
		"New": mi.newEnvironment,
	}}
}

func (mi *ModuleInstance) newEnvironment(c goja.ConstructorCall) *goja.Object {
	rt := mi.vu.Runtime()

	var opts environment.JSOptions
	if err := rt.ExportTo(c.Argument(0), &opts); err != nil {
		common.Throw(rt, errors.New("unable to parse options"))
	}

	test, err := fs.FindTest(opts.Source)
	if err != nil {
		common.Throw(rt, fmt.Errorf("Cannot find the test in %s", opts.Source))
	}
	if err := test.ReadOptions(); err != nil {
		common.Throw(rt, fmt.Errorf("Cannot read options ini"))
	}

	env := environment.NewEnvironment(test, nil)

	env.VU = mi.vu
	env.JSOptions = opts

	// FIXME this should be config path being passed in...
	env.ParentContext, err = kubernetes.CurrentContext("")
	if err != nil {
		common.Throw(rt, err)
	}

	env.TestName(environment.TestName(env.Test.FolderName()))

	environment.InjectInitEnv(env)

	return rt.ToValue(env).ToObject(rt)
}
