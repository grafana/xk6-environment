package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"xk6-environment/pkg/fs"

	k6crd "github.com/grafana/k6-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8Yaml "k8s.io/apimachinery/pkg/util/yaml"
)

// waitCondition is meant to help decide what is the end of the test.
type waitCondition struct {
	// what was configured
	// is condition on the test execution as a whole?
	// e.g. in the future it could be IsThreshold
	isTestExecution bool
	conditionKind   string // finished | error

	isTimeout bool
	duration  time.Duration

	// k8s resource
	kind, name, namespace string
	condF                 func(context.Context) (done bool, err error)
}

func NewWaitCondition(s string) (*waitCondition, error) {
	var wc waitCondition

	// TODO: add validation
	ss := strings.Split(s, "=")
	switch ss[0] {
	case "test":
		wc.isTestExecution = true
		wc.conditionKind = ss[1]
		break
	case "timeout":
		wc.isTimeout = true
		if d, err := time.ParseDuration(ss[1]); err != nil {
			return nil, err
		} else {
			wc.duration = d
		}
	}

	return &wc, nil
}

// wait conditon must be applied to specific test to be usable
func (wc *waitCondition) Apply(c *Client, testName string, td fs.TestDef) error {
	wc.name = testName

	// TODO: refactor this many-IFs monster!

	if wc.isTestExecution {
		if td.IsYaml() {
			// k6-operator mode

			// we'll need to parse spec to get name and namespace
			rawSpec, err := td.ReadTest()
			if err != nil {
				return err
			}

			var crdSpec k6crd.K6
			dec := k8Yaml.NewYAMLOrJSONDecoder(bytes.NewReader(rawSpec), 1000)
			if err := dec.Decode(&crdSpec); err != nil {
				return err
			}
			if len(crdSpec.Namespace) == 0 {
				crdSpec.Namespace = "default"
			}

			wc.kind = "K6"
			wc.namespace = crdSpec.Namespace

			wc.condF = func(ctx context.Context) (done bool, err error) {
				// Why not subscribe to events here: K6 CRD for instance does not even have
				// events yet. So polling state instead: even if we add events to K6 CRD tomorrow,
				// it'd be less stable than state. Consider other options.
				// OTOH, events might be an option for Job (k6-standalone mode) as it's a builtin resource.

				// /apis/k6.io/v1alpha1/k6s - to get K6List
				// /apis/k6.io/v1alpha1/namespaces/default/k6s/k6-sample - to get K6
				// Ref: https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-uris
				d, err := c.clientset.RESTClient().Get().AbsPath(
					fmt.Sprintf("/apis/k6.io/v1alpha1/namespaces/%s/k6s/%s", wc.namespace, crdSpec.Name),
				).DoRaw(ctx)
				if err != nil {
					return false, err
				}
				var k6 k6crd.K6
				if err := json.Unmarshal(d, &k6); err != nil {
					return false, err
				}

				if wc.conditionKind == "finished" {
					if k6.Status.Stage == "finished" {
						return true, nil
					}
				} else {
					if k6.Status.Stage == "error" {
						return true, nil
					}
				}

				return false, nil
			}
		} else {
			// k6-standalone mode

			wc.kind = "Job"
			wc.namespace = "default"

			wc.condF = func(ctx context.Context) (done bool, err error) {
				job, err := c.clientset.BatchV1().Jobs(wc.namespace).Get(ctx, wc.name, metav1.GetOptions{})
				if err != nil {
					return false, err
				}

				if job.Status.Active > 0 {
					return false, nil
				}

				if wc.conditionKind == "finished" {
					if job.Status.Succeeded > 0 {
						return true, nil
					}
				} else {
					if job.Status.Failed > 0 {
						return true, nil
					}
				}

				return false, nil
			}
		}
	}

	return nil
}
