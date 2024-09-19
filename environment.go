// Package environment provides functionality for xk6-environment extension.
package environment

import (
	"fmt"
	"time"

	"github.com/grafana/xk6-environment/pkg/environment"
	"github.com/grafana/xk6-environment/pkg/fs"
	"github.com/grafana/xk6-environment/pkg/kubernetes"

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

func (mod *goModuleImpl) newEnvironment(params interface{}) (goEnvironment, error) {
	// the only implementation supported now is vcluster so
	// omitting the parameter here for simplicity
	name, _, initFolder, err := processParams(params)
	if err != nil {
		return nil, err
	}

	// the folder might be empty so skip it
	// TODO: add some logging here?
	fenv, err := fs.FindEnv(initFolder)
	if err != nil {
		fmt.Println("FindTest: ", err)
	}

	env := environment.NewEnvironment(fenv, nil)
	env.JSOptions = environment.JSOptions{
		Source: initFolder,
	}

	env.SetTestName(name)

	return goEnvironmentImpl{
		e:  env,
		vu: mod.vu,
	}, nil
}

func (mod *goModuleImpl) defaultEnvironmentGetter() (goEnvironment, error) {
	return mod.goEnvironment, nil
}

type goEnvironmentImpl struct {
	e  *environment.Environment
	vu modules.VU
}

var _ goEnvironment = (*goEnvironmentImpl)(nil)

// initMethod is the go representation of the create method.
//
//nolint:nilnil,nilerr
func (impl goEnvironmentImpl) initMethod() (interface{}, error) {
	if err := impl.e.Create(impl.vu.Context()); err != nil {
		return err.Error(), nil
	}

	return nil, nil
}

// deleteMethod is the go representation of the delete method.
//
//nolint:nilnil,nilerr
func (impl goEnvironmentImpl) deleteMethod() (interface{}, error) {
	if err := impl.e.Delete(impl.vu.Context()); err != nil {
		return err.Error(), nil
	}

	return nil, nil
}

// applyMethod is the go representation of the apply method.
//
//nolint:nilnil,nilerr
func (impl goEnvironmentImpl) applyMethod(fileArg string) (interface{}, error) {
	if err := impl.e.Apply(impl.vu.Context(), fileArg); err != nil {
		return err.Error(), nil
	}

	return nil, nil
}

// applySpecMethod is the go representation of the applySpec method.
//
//nolint:nilnil,nilerr
func (impl goEnvironmentImpl) applySpecMethod(specArg string) (interface{}, error) {
	if err := impl.e.ApplySpec(impl.vu.Context(), specArg); err != nil {
		return err.Error(), nil
	}

	return nil, nil
}

//nolint:nilnil,nilerr
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

	if err := impl.e.Wait(impl.vu.Context(), wc); err != nil {
		return err.Error(), nil
	}

	return nil, nil
}

func (impl goEnvironmentImpl) getNMethod(typeArg string, optsArg interface{}) (float64, error) {
	if typeArg != "pods" {
		// TODO remove this once error propagation works
		fmt.Println("got error: only pods are currently supported")
		return 0, nil
	}
	opts := map[string]interface{}{}
	if optsArg != nil {
		var ok bool
		opts, ok = optsArg.(map[string]interface{})
		if !ok {
			err := fmt.Errorf(
				`2nd argument in getN() must be an object of the form {"namespace":"ns","label": "selector"}, got: %+v`,
				optsArg)
			// TODO remove this once error propagation works
			fmt.Println("got error", err)
			return 0, nil
		}
	}

	n, err := impl.e.GetN(impl.vu.Context(), opts)
	if err != nil {
		// TODO remove this once error propagation works
		fmt.Println("got error", err)
		return 0, nil
	}
	return float64(n), nil
}

// TODO: tygor issue for this boilerplate
func processParams(paramsArg interface{}) (name, implementation, initFolder string, err error) {
	e := fmt.Errorf(`Environment() expects an object; got: %+v`, paramsArg)
	params, ok := paramsArg.(map[string]interface{})
	if !ok {
		err = e
		return
	}

	name, _ = params["name"].(string)
	implementation, _ = params["implementation"].(string)
	initFolder, _ = params["initFolder"].(string)

	return
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
