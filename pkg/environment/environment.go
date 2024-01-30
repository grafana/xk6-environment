package environment

import (
	"context"
	"fmt"
	"os"
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

type CriteriaDef struct {
	Test      string
	TimeLimit string
	Event     string
	Loki      string
}

type JSOptions struct {
	Source         string
	IncludeGrafana bool // not supported yet
	Criteria       CriteriaDef
	Timeout        string // not supported yet: should be passed to waiting functions
}

func (opts *JSOptions) getCondition() string {
	if len(opts.Criteria.Test) > 0 {
		return fmt.Sprintf("test=%s", opts.Criteria.Test)
	}
	if len(opts.Criteria.TimeLimit) > 0 {
		return fmt.Sprintf("timeout=%s", opts.Criteria.TimeLimit)
	}

	// not supported yet - don't use it!
	if len(opts.Criteria.Event) > 0 {
		return opts.Criteria.Event
	}
	return opts.Criteria.Loki
}

// Environment is the type for our custom API.
type Environment struct {
	VU modules.VU

	opts             *options
	kubernetesClient *kubernetes.Client
	// This is now set from init context, see newEnvironment call
	ParentContext string
	TestName      string
	Test          *fs.Test

	// set from JS
	JSOptions

	// This is from k6-environment CLI:
	// no logging is happening at the level of Environment here.
	logger *zap.Logger
}

func NewEnvironment(test *fs.Test, logger *zap.Logger) *Environment {
	return &Environment{
		opts: &options{
			"",
		},
		kubernetesClient: nil,
		ParentContext:    "",
		TestName:         "",
		Test:             test,

		logger: logger,
	}
}

func (e *Environment) SetTestName(n string) {
	e.TestName = n
}

func (e *Environment) InitKubernetes(ctx context.Context, ctxName string) (err error) {
	if ctxName != "" {
		if err := kubernetes.SetContext(e.opts.ConfigPath, ctxName); err != nil {
			return err
		}
	}

	e.kubernetesClient, err = kubernetes.NewClient(ctx, e.opts.ConfigPath)
	return
}

func (e *Environment) getParent(ctx context.Context) (err error) {
	e.ParentContext, err = kubernetes.CurrentContext("")
	return
}

// In the end of each k6 lifecycle step, we should go back to parent
// Kubernetes context, in order to have a clean state to start with
func (e *Environment) parent(ctx context.Context) error {
	return kubernetes.SetContext(e.opts.ConfigPath, e.ParentContext)
}

func (e *Environment) Describe() string {
	return fmt.Sprintf(`Test name: %s, with files: %+v. 
						jsopts: %v,
						K8s parent context: %s\n`,
		e.TestName, e.Test, e.JSOptions, e.ParentContext)
}

// to be called in setup()
func (e *Environment) Create(ctx context.Context) error {
	if err := e.getParent(ctx); err != nil {
		return err
	}

	if err := vcluster.Create(e.TestName); err != nil {
		return err
	}

	if err := e.InitKubernetes(ctx, e.TestName); err != nil {
		return fmt.Errorf("unable to initialize Kubernetes client: %w", err)
	}

	if err := e.kubernetesClient.Deploy(e.Test.Kubernetes); err != nil {
		return err
	}

	return e.parent(ctx)
}

// to be called in teardown()
func (e *Environment) Delete(ctx context.Context) error {
	// This will be needed if / when vcluster is done via Helm
	// if err := e.InitKubernetes(ctx, ""); err != nil {
	// 	return fmt.Errorf("unable to initialize Kubernetes client: %w", err)
	// }

	if err := vcluster.Delete(e.TestName); err != nil {
		return err
	}

	return kubernetes.DeleteContext(e.opts.ConfigPath, e.TestName)
}

func (e *Environment) RunTest(ctx context.Context) error {
	if err := e.kubernetesClient.CreateTest(ctx, e.TestName, e.Test.Def); err != nil {
		return err
	}

	wc, err := kubernetes.NewWaitCondition(e.JSOptions.getCondition())
	if err != nil {
		return err
	}

	if err := e.Wait(ctx, wc); err != nil {
		return err
	}

	return nil
}

// non-public in xk6-environment
func (e *Environment) Wait(ctx context.Context, wc *kubernetes.WaitCondition) error {
	if err := e.getParent(ctx); err != nil {
		return err
	}

	if err := e.InitKubernetes(ctx, e.TestName); err != nil {
		return fmt.Errorf("unable to initialize Kubernetes client: %w", err)
	}

	if err := wc.Apply(e.kubernetesClient, e.TestName, e.Test.Def); err != nil {
		return err
	}
	if err := e.kubernetesClient.Wait(ctx, wc); err != nil {
		return err
	}

	return e.parent(ctx)
}

// currently unused
func newTestName(prefix string) string {
	t := time.Now()
	return prefix + t.Format("-060102-150405")
}

func (e *Environment) Apply(ctx context.Context, file string) error {
	if err := e.getParent(ctx); err != nil {
		return err
	}

	if err := e.InitKubernetes(ctx, e.TestName); err != nil {
		return fmt.Errorf("unable to initialize Kubernetes client: %w", err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	if err = e.kubernetesClient.Apply(string(data)); err != nil {
		return err
	}

	return e.parent(ctx)
}

func (e *Environment) ApplySpec(ctx context.Context, spec interface{}) error {
	// TODO
	// o, err := e.kubernetesClient.Create(spec)
	return nil //err
}
