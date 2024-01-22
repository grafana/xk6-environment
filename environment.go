package environment

import (
	"fmt"
	"xk6-environment/pkg/environment"
	"xk6-environment/pkg/fs"
	"xk6-environment/pkg/kubernetes"

	"go.k6.io/k6/js/modules"
)

//go:generate go run github.com/szkiba/tygor@latest --package environment --skeleton index.d.ts
//go:generate go run github.com/szkiba/tygor@latest doc --inject README.md index.d.ts

func init() {
	register(newModule)
}

func newModule(vu modules.VU) goModule {
	return &goModuleImpl{
		vu:            vu,
		goEnvironment: &goEnvironmentImpl{},
	}
}

type goModuleImpl struct {
	vu            modules.VU
	goEnvironment goEnvironment
}

var _ goModule = (*goModuleImpl)(nil)

func (mod *goModuleImpl) newEnvironment(name, iType, initFolder string) (goEnvironment, error) {
	// msg := fmt.Sprintf("Hello, %s!", nameArg)
	// rt := mod.vu.Runtime()

	opts := environment.JSOptions{
		Source: initFolder,
	}

	test, err := fs.FindTest(opts.Source)
	if err != nil {
		return nil, fmt.Errorf("Cannot find the test in %s", opts.Source)
	}
	if err := test.ReadOptions(); err != nil {
		return nil, fmt.Errorf("Cannot read options ini")
	}

	env := environment.NewEnvironment(test, nil)

	// env.VU = mi.vu
	env.JSOptions = opts

	// FIXME this should be config path being passed in...
	env.ParentContext, err = kubernetes.CurrentContext("")
	if err != nil {
		return nil, err
	}

	env.TestName(environment.TestName(env.Test.FolderName()))

	environment.InjectInitEnv(env)

	return goEnvironmentImpl{
		Environment: env,
		vu:          mod.vu,
	}, nil
}

func (mod *goModuleImpl) defaultEnvironmentGetter() (goEnvironment, error) {
	return mod.goEnvironment, nil
}

type goEnvironmentImpl struct {
	*environment.Environment
	vu modules.VU
}

var _ goEnvironment = (*goEnvironmentImpl)(nil)

// initMethod is the go representation of the create method.
func (impl goEnvironmentImpl) initMethod() error {
	return impl.Create(impl.vu.Context())
}

// runTestMethod is the go representation of the runTest method.
func (impl goEnvironmentImpl) runTestMethod() error {
	return impl.RunTest(impl.vu.Context())
}

// deleteMethod is the go representation of the delete method.
func (impl goEnvironmentImpl) deleteMethod() error {
	return impl.Delete(impl.vu.Context())
}

// applyMethod is the go representation of the apply method.
func (impl goEnvironmentImpl) applyMethod(fileArg string) error {
	return impl.Apply(impl.vu.Context(), fileArg)
}

// applySpecMethod is the go representation of the applySpec method.
func (impl goEnvironmentImpl) applySpecMethod(specArg interface{}) error {
	return impl.ApplySpec(impl.vu.Context(), specArg)
}

func (impl goEnvironmentImpl) waitMethod(objArg interface{}) error {
	waitOptions, ok := objArg.(map[string]interface{})
	if !ok {
		return fmt.Errorf("wait() requires an object that can be converted to map[string]interface{}, got: %+v", objArg)
	}
	// only events here now
	if wc, ok := isEvent(waitOptions); ok {
		return impl.Wait(impl.vu.Context(), &wc)
	}
	return fmt.Errorf("wait() requires an object that can be converted to a supported wait condition, got: %+v", waitOptions)
}

func isEvent(waitOptions map[string]interface{}) (wc kubernetes.WaitCondition, ok bool) {
	if wc.Kind, ok = waitOptions["type"].(string); !ok {
		return
	}

	if wc.Name, ok = waitOptions["name"].(string); !ok {
		return
	}

	if wc.Namespace, ok = waitOptions["namespace"].(string); !ok {
		return
	}

	if wc.Reason, ok = waitOptions["reason"].(string); !ok {
		return
	}

	return
}
