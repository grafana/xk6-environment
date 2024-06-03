package environment

import (
	"fmt"
	"time"
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

	env.JSOptions = opts

	env.SetTestName(env.Test.FolderName())

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
func (impl goEnvironmentImpl) initMethod() (interface{}, error) {
	return impl.Create(impl.vu.Context()), nil
}

// runTestMethod is the go representation of the runTest method.
func (impl goEnvironmentImpl) runTestMethod() error {
	return impl.RunTest(impl.vu.Context())
}

// deleteMethod is the go representation of the delete method.
func (impl goEnvironmentImpl) deleteMethod() (interface{}, error) {
	return impl.Delete(impl.vu.Context()), nil
}

// applyMethod is the go representation of the apply method.
func (impl goEnvironmentImpl) applyMethod(fileArg string) error {
	return impl.Apply(impl.vu.Context(), fileArg)
}

// applySpecMethod is the go representation of the applySpec method.
func (impl goEnvironmentImpl) applySpecMethod(specArg string) error {
	return impl.ApplySpec(impl.vu.Context(), specArg)
}

func (impl goEnvironmentImpl) waitMethod(conditionArg interface{}, optsArg interface{}) (interface{}, error) {
	wc, err := kubernetes.NewWaitCondition(conditionArg)
	if err != nil {
		// this is a syntax error in definition of condition itself
		return err.Error(), nil
	}

	if optsArg != nil {
		interval, timeout, err := waitOptions(optsArg)
		if err != nil {
			// this is a syntax error in options
			return err.Error(), nil
		}
		wc.TimeParams(interval, timeout)
	}

	wc.Build()

	return impl.Wait(impl.vu.Context(), wc), nil
}

func waitOptions(optsArg interface{}) (interval, timeout time.Duration, err error) {
	e := fmt.Errorf(`2nd argument in wait() must be an object of the form {interval:"1h",timeout:"5m"}; got: %+v`, optsArg)
	opts, ok := optsArg.(map[string]interface{})
	if !ok {
		err = e
		return
	}

	intervalS, _ := opts["interval"].(string)
	timeoutS, _ := opts["timeout"].(string)

	if len(intervalS) > 0 {
		interval, err = time.ParseDuration(intervalS)
		if err != nil {
			return
		}
	}

	if len(timeoutS) > 0 {
		timeout, err = time.ParseDuration(timeoutS)
	}

	return
}
