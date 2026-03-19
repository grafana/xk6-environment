// Package environment provides functionality for xk6-environment extension.
package environment

import (
	"fmt"
	"time"

	"github.com/grafana/sobek"
	"github.com/grafana/xk6-environment/pkg/environment"
	"github.com/grafana/xk6-environment/pkg/fs"
	"github.com/grafana/xk6-environment/pkg/kubernetes"

	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/environment", new(rootModule))
}

type rootModule struct{}

func (*rootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &moduleInstance{vu: vu}
}

var _ modules.Module = (*rootModule)(nil)

type moduleInstance struct {
	vu modules.VU
}

func (m *moduleInstance) Exports() modules.Exports {
	rt := m.vu.Runtime()
	defaultObj := rt.NewObject()
	bindEnvMethods(rt, defaultObj, &goEnvironmentImpl{vu: m.vu})

	return modules.Exports{
		Named: map[string]any{
			"Environment": m.newEnvironmentConstructor,
		},
		Default: defaultObj,
	}
}

var _ modules.Instance = (*moduleInstance)(nil)

func (m *moduleInstance) newEnvironmentConstructor(call sobek.ConstructorCall) *sobek.Object {
	val := call.Argument(0)
	if sobek.IsNaN(val) || sobek.IsUndefined(val) {
		panic("can't get a constructor argument")
	}

	// the only implementation supported now is vcluster so
	// omitting the parameter here for simplicity

	name, _, initFolder, err := processParams(val.Export())
	if err != nil {
		panic(m.vu.Runtime().NewTypeError(err.Error()))
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

	impl := &goEnvironmentImpl{e: env, vu: m.vu}
	bindEnvMethods(m.vu.Runtime(), call.This, impl)

	return nil
}

func bindEnvMethods(rt *sobek.Runtime, obj *sobek.Object, impl *goEnvironmentImpl) {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	toValue := rt.ToValue

	must(obj.Set("init", toValue(impl.initMethod)))
	must(obj.Set("delete", toValue(impl.deleteMethod)))
	must(obj.Set("apply", toValue(impl.applyMethod)))
	must(obj.Set("applySpec", toValue(impl.applySpecMethod)))
	must(obj.Set("wait", toValue(impl.waitMethod)))
	must(obj.Set("getN", toValue(impl.getNMethod)))
}

type goEnvironmentImpl struct {
	e  *environment.Environment
	vu modules.VU
}

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
