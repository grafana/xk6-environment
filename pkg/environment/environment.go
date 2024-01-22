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

var (
	// we need to store Environment initialized in init context
	// and setup() so that we can restore it during main VU execution
	// TODO: this could use some cleaning up...
	initEnvironment *Environment
	// this is a bit hacky but it appears that init context is
	// re-instantiating object more than once
	injectCalled bool
	// This mess can be avoided with some proper refactoring done:
	// inject data in some object not connected to Environment
	// and read that whenever needed.
	// Perhaps, there should be:
	// - Environment as JS interface
	// - Environment as internal implementation which can be shared with CLI tool.
)

func InjectInitEnv(env *Environment) {
	// allow injection only the first time
	if injectCalled == false {
		initEnvironment = env
		injectCalled = true
	}
}

// get configs from init env
func (e *Environment) getEnvDataFromInit() {
	e.opts = initEnvironment.opts
	e.ParentContext = initEnvironment.ParentContext
	e.testName = initEnvironment.testName
	e.Test = initEnvironment.Test
	e.JSOptions = initEnvironment.JSOptions
}

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
	testName      string
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
		testName:         "",
		Test:             test,

		logger: logger,
	}
}

func (e *Environment) TestName(n string) {
	e.testName = n
}

func (e *Environment) initKubernetesClient(ctx context.Context, ctxName string) (err error) {
	e.kubernetesClient, err = kubernetes.NewClient(ctx, e.opts.ConfigPath, ctxName)
	return
}

func (e *Environment) Describe() string {
	return fmt.Sprintf("Test name: `%s`, with files: %+v. K8s parent context is `%s`\n", e.testName, e.Test, e.ParentContext)
}

// adapted from k6-environment
func (e *Environment) Create(ctx context.Context) error {
	if err := vcluster.Create(e.testName); err != nil {
		return err
	}

	if err := e.initKubernetesClient(ctx, e.testName); err != nil {
		return fmt.Errorf("unable to initialize Kubernetes client: %w", err)
	}

	if err := e.kubernetesClient.Deploy(e.Test.Kubernetes); err != nil {
		return err
	}

	return nil
}

// adapted from k6-environment
func (e *Environment) Delete(ctx context.Context) error {
	e.getEnvDataFromInit()

	if err := e.initKubernetesClient(ctx, e.ParentContext); err != nil {
		return fmt.Errorf("unable to initialize Kubernetes client: %w", err)
	}

	if err := vcluster.Delete(e.testName); err != nil {
		return err
	}

	return nil
}

func (e *Environment) RunTest(ctx context.Context) error {
	e.getEnvDataFromInit()

	// context was set during creation of environment
	// so its fine to pass empty string here
	if err := e.initKubernetesClient(ctx, ""); err != nil {
		return fmt.Errorf("unable to initialize Kubernetes client: %w", err)
	}

	if err := e.kubernetesClient.CreateTest(ctx, e.testName, e.Test.Def); err != nil {
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
	if err := wc.Apply(e.kubernetesClient, e.testName, e.Test.Def); err != nil {
		return err
	}
	return e.kubernetesClient.Wait(ctx, wc)
}

func TestName(prefix string) string {
	t := time.Now()
	return prefix + t.Format("-060102-150405")
}

func (e *Environment) Apply(ctx context.Context, file string) error {
	e.getEnvDataFromInit()

	// context was set during creation of environment
	// so its fine to pass empty string here
	if err := e.initKubernetesClient(ctx, ""); err != nil {
		return fmt.Errorf("unable to initialize Kubernetes client: %w", err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	return e.kubernetesClient.Apply(string(data))
}

func (e *Environment) ApplySpec(ctx context.Context, spec interface{}) error {
	e.getEnvDataFromInit()

	// context was set during creation of environment
	// so its fine to pass empty string here
	if err := e.initKubernetesClient(ctx, ""); err != nil {
		return fmt.Errorf("unable to initialize Kubernetes client: %w", err)
	}

	// TODO
	// o, err := e.kubernetesClient.Create(spec)
	return nil //err
}
