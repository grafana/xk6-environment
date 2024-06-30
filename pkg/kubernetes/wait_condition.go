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
	if status, ok := waitOptions["value"].(string); ok {
		wc.Status = metav1.ConditionStatus(status)
	}
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

func (wc *WaitCondition) crdStatusCondition() {
	wc.condF = func(c *Client) func(ctx context.Context) (done bool, err error) {
		return func(ctx context.Context) (done bool, err error) {
			// we don't know when CRD would be first created, so we should
			// look up GVK whenever a new wait condition is required
			gvk, err := kindToGVK(wc.Kind, c.discoveryClient)
			if err != nil {
				return false, err
			}

			// we need resource to be able to query dynamic client
			restMapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				return false, err
			}

			unstructured, err := c.dynamicClient.Resource(restMapping.Resource).Namespace(wc.Namespace).Get(ctx, wc.Name, metav1.GetOptions{})
			if err != nil {
				// From here on, we try to wait until resource reaches required state,
				// so don't return this error.
				// TODO add some logging here?
				return false, nil
			}

			c, found, err := metav1u.NestedSlice(unstructured.UnstructuredContent(), "status", "conditions")
			if err != nil {
				// conversion error should be returned
				return false, err
			}
			if !found {
				// Resource is without conditions: wait more in case
				// its conditions change.
				return false, nil
			}

			cond := meta.FindStatusCondition(getConditions(c), wc.ConditionType)
			if cond != nil && cond.Status == wc.Status {
				return true, nil
			}

			return false, nil
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
