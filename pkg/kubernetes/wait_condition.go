package kubernetes

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1u "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// WaitCondition indicates how long to wait
type WaitCondition struct {
	// what was configured
	// is condition on the test execution as a whole?
	// e.g. in the future it could be IsThreshold
	isTestExecution bool
	conditionKind   string // finished | error

	interval, timeout time.Duration

	Resource // what resource to watch
	State    // we wait until certain state

	condF func(*Client) func(context.Context) (done bool, err error)
}

type Resource struct {
	Kind, Name, Namespace string
}

type State struct {
	StateType

	// for events
	Reason string

	// for .status conditions
	Status        metav1.ConditionStatus
	ConditionType string

	// for logs
	// Log               string
}

type StateType int

const (
	Invalid StateType = iota
	Event
	Status
)

func NewWaitCondition(conditionArg interface{}) (wc *WaitCondition, err error) {
	waitOptions, ok := conditionArg.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("wait() requires an object that can be converted to map[string]interface{}, got: %+v", conditionArg)
		return
	}
	wc = &WaitCondition{}
	// set defaults
	wc.interval, wc.timeout = 2*time.Second, 1*time.Hour

	// extract whatever possible
	wc.Kind, _ = waitOptions["kind"].(string)
	wc.Name, _ = waitOptions["name"].(string)
	wc.Namespace, _ = waitOptions["namespace"].(string)
	wc.Reason, _ = waitOptions["reason"].(string)
	if status, ok := waitOptions["status"].(string); ok {
		wc.Status = metav1.ConditionStatus(status)
	}
	// wc.Value, _ = waitOptions["value"].(string)
	wc.ConditionType, _ = waitOptions["condition_type"].(string)

	wc.DeriveType()
	if !wc.Validate() {
		return nil, fmt.Errorf("format of condition for wait() is invalid; refer to documentation")
	}
	return
}

func (wc *WaitCondition) DeriveType() {
	if len(wc.Reason) > 0 {
		wc.StateType = Event
	} else if len(wc.ConditionType) > 0 || len(wc.Status) > 0 {
		wc.StateType = Status
	} else {
		wc.StateType = Invalid
	}
}

func (wc *WaitCondition) Validate() bool {
	return wc.StateType > Invalid &&
		len(wc.Kind) > 0 && len(wc.Namespace) > 0 && len(wc.Name) > 0
}

func (wc *WaitCondition) TimeParams(interval, timeout time.Duration) {
	wc.interval, wc.timeout = interval, timeout
}

func (wc *WaitCondition) Build() {
	fmt.Println("Build", wc, wc.StateType)
	switch wc.StateType {
	case Status:
		wc.crdStatusCondition()

	default: // == Event
		wc.eventCondition()
	}
}

/*
// wait conditon must be applied to specific test to be usable

	func (wc *WaitCondition) Apply(c *Client, testName string, td fs.TestDef) error {
		// TODO: refactor this many-IFs monster!

		if len(wc.Reason) > 0 {
			return wc.eventWaiter(c)
		}

		wc.Name = testName

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
				wc.Kind = crdSpec.Kind
				wc.Namespace = crdSpec.Namespace

				wc.condF = func(ctx context.Context) (done bool, err error) {
					// Why not subscribe to events here: K6 CRD for instance does not even have
					// events yet. So polling state instead: even if we add events to K6 CRD tomorrow,
					// it'd be less stable than state.
					// OTOH, events might be an option for Job (k6-standalone mode) as it's a builtin resource.

					// Figuring out URIs:
					// /apis/k6.io/v1alpha1/k6s - to get K6List
					// /apis/k6.io/v1alpha1/namespaces/default/k6s/k6-sample - to get K6
					// Ref: https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-uris
					crdList := "testruns"
					if wc.Kind == "K6" {
						crdList = "k6s"
					}

					d, err := c.clientset.RESTClient().Get().AbsPath(
						fmt.Sprintf("/apis/k6.io/v1alpha1/namespaces/%s/%s/%s", wc.Namespace, crdList, crdSpec.Name),
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

				wc.Kind = "Job"
				wc.Namespace = "default"

				wc.condF = func(ctx context.Context) (done bool, err error) {
					job, err := c.clientset.BatchV1().Jobs(wc.Namespace).Get(ctx, wc.Name, metav1.GetOptions{})
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
*/
func (wc *WaitCondition) crdStatusCondition() {
	wc.condF = func(c *Client) func(ctx context.Context) (done bool, err error) {
		return func(ctx context.Context) (done bool, err error) {
			// we don't know when CRD would be first created, so we should
			// look up GVK whenever a new wait condition is required
			gvk, err := kindToGVK(wc.Kind, c.discoveryClient)
			// fmt.Println("gvk", wc.Kind, gvk, err)
			if err != nil {
				return false, err
			}

			// we need resource to be able to query dynamic client
			restMapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			// fmt.Println(restMapping, err)
			if err != nil {
				return false, err
			}

			unstructured, err := c.dynamicClient.Resource(restMapping.Resource).Namespace(wc.Namespace).Get(ctx, wc.Name, metav1.GetOptions{})
			if err != nil {
				return
			}
			// fmt.Println(gvk, restMapping)

			c, found, err := metav1u.NestedSlice(unstructured.UnstructuredContent(), "status", "conditions")
			// fmt.Println(c, found, err)
			if err != nil {
				return false, err
			}
			if !found {
				return false, fmt.Errorf("resource %s without conditions", wc.Name)
			}

			cond := meta.FindStatusCondition(getConditions(c), wc.ConditionType)
			if cond != nil && cond.Status == wc.Status {
				return true, nil
			}

			return
		}
	}
}

func (wc *WaitCondition) eventCondition() {
	wc.condF = func(c *Client) func(ctx context.Context) (done bool, err error) {
		return func(ctx context.Context) (done bool, err error) {
			events, err := c.clientset.CoreV1().Events(wc.Namespace).List(ctx, metav1.ListOptions{
				TypeMeta: metav1.TypeMeta{
					Kind: wc.Kind,
				},
				FieldSelector: "involvedObject.name=" + wc.Name,
			})
			if err != nil {
				return
			}

			for _, event := range events.Items {
				if event.Reason == wc.Reason {
					done = true
					return
				}
			}

			return
		}
	}
}