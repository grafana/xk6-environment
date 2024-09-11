// Package environment contains the main implementation for
// Environment class.
package environment

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"xk6-environment/pkg/fs"
	"xk6-environment/pkg/kubernetes"
	"xk6-environment/pkg/vcluster"

	"go.k6.io/k6/js/modules"
	"go.uber.org/zap"
)

// Options for environment
type options struct {
	// k8s
	ConfigPath string
}

type criteriaDef struct {
	Test      string
	TimeLimit string
	Event     string
	Loki      string
}

// JSOptions holds configuration of environment,
// as specified by a user in the script.
type JSOptions struct {
	Source         string
	IncludeGrafana bool // not supported yet
	Criteria       criteriaDef
	Timeout        string // not supported yet: should be passed to waiting functions
}

// Environment is the type for our custom API.
type Environment struct {
	VU modules.VU

	opts             *options
	kubernetesClient *kubernetes.Client
	// This is now set from init context, see newEnvironment call
	ParentContext string
	TestName      string
	envDesc       *fs.EnvDescription

	// set from JS
	JSOptions

	// This is from k6-environment CLI:
	// no logging is happening at the level of Environment here.
	logger *zap.Logger
}

// NewEnvironment constructs a new Environment.
func NewEnvironment(fenv *fs.EnvDescription, logger *zap.Logger) *Environment {
	return &Environment{
		opts: &options{
			"",
		},
		kubernetesClient: nil,
		ParentContext:    "",
		TestName:         "",
		envDesc:          fenv,

		logger: logger,
	}
}

// SetTestName set the name of environment.
func (e *Environment) SetTestName(n string) {
	e.TestName = n
}

// InitKubernetes switches to the given Kubernetes context if provided and
// builds a new Kubernetes client.
func (e *Environment) InitKubernetes(ctx context.Context, ctxName string) (err error) {
	if ctxName != "" {
		if err := kubernetes.SetContext(e.opts.ConfigPath, ctxName); err != nil {
			return err
		}
	}

	e.kubernetesClient, err = kubernetes.NewClient(ctx, e.opts.ConfigPath)
	return
}

func (e *Environment) getParent(_ context.Context) (err error) {
	e.ParentContext, err = kubernetes.CurrentContext("")
	return
}

// In the end of each k6 lifecycle step, we should go back to parent
// Kubernetes context, in order to have a clean state to start with
func (e *Environment) parent(_ context.Context) error {
	return kubernetes.SetContext(e.opts.ConfigPath, e.ParentContext)
}

// Describe returns a short text description of the Environment.
func (e *Environment) Describe() string {
	return fmt.Sprintf(`Test name: %s, with files: %+v. 
						jsopts: %v,
						K8s parent context: %s\n`,
		e.TestName, e.envDesc, e.JSOptions, e.ParentContext)
}

// Create creates a vcluster and deploys the initial environment
// according to user's configuration.
// Create is meant to be called in setup() of the script.
func (e *Environment) Create(ctx context.Context) (err error) {
	if err = e.getParent(ctx); err != nil {
		return err
	}

	// always return to parent context so that
	// the next operations can continue
	defer func() {
		e := e.parent(ctx)
		// overwrite return value, only if it's nil;
		// otherwise, return the error from main function
		if err == nil {
			err = e
		}
	}()

	if err = vcluster.Create(e.TestName); err != nil {
		return
	}

	if err = e.InitKubernetes(ctx, e.TestName); err != nil {
		return fmt.Errorf("unable to initialize Kubernetes client: %w", err)
	}

	err = e.kubernetesClient.Deploy(ctx, e.envDesc)
	return
}

// Delete deletes a vcluster.
// Delete is meant to be called in teardown() of the script.
func (e *Environment) Delete(_ context.Context) error {
	// This will be needed if / when vcluster is done via Helm
	// if err := e.InitKubernetes(ctx, ""); err != nil {
	// 	return fmt.Errorf("unable to initialize Kubernetes client: %w", err)
	// }

	if err := vcluster.Delete(e.TestName); err != nil {
		return err
	}

	return kubernetes.DeleteContext(e.opts.ConfigPath, e.TestName)
}

// Wait blocks execution until given wait condition is reached.
func (e *Environment) Wait(ctx context.Context, wc *kubernetes.WaitCondition) (err error) {
	if err = e.getParent(ctx); err != nil {
		return
	}
	defer func() {
		e := e.parent(ctx)
		// overwrite return value, only if it's nil;
		// otherwise, return the error from main function
		if err == nil {
			err = e
		}
	}()

	if err = e.InitKubernetes(ctx, e.TestName); err != nil {
		return fmt.Errorf("unable to initialize Kubernetes client: %w", err)
	}

	err = e.kubernetesClient.Wait(ctx, wc)
	return
}

// newTestName currently unused
//
//nolint:unused
func newTestName(prefix string) string {
	t := time.Now()
	return prefix + t.Format("-060102-150405")
}

// Apply deploys the manifest file.
func (e *Environment) Apply(ctx context.Context, file string) error {
	//nolint:forbidigo
	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return err
	}
	return e.ApplySpec(ctx, string(data))
}

// ApplySpec deploys the manifest spec.
func (e *Environment) ApplySpec(ctx context.Context, spec string) (err error) {
	if err = e.getParent(ctx); err != nil {
		return
	}
	defer func() {
		e := e.parent(ctx)
		// overwrite return value, only if it's nil;
		// otherwise, return the error from main function
		if err == nil {
			err = e
		}
	}()

	if err = e.InitKubernetes(ctx, e.TestName); err != nil {
		return fmt.Errorf("unable to initialize Kubernetes client: %w", err)
	}

	err = e.kubernetesClient.Apply(ctx, bytes.NewBufferString(spec))
	return err
}
