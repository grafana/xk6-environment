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
	interval, timeout time.Duration

	resource // what resource to watch
	state    // we wait until certain state

	condF func(*Client) func(context.Context) (done bool, err error)
}

type resource struct {
	Kind, Name, Namespace string
}

type state struct {
	stateType

	// for events
	Reason string

	// for .status.conditions
	Status        metav1.ConditionStatus // "value"
	ConditionType string

	// for .status custom values
	StatusKey   string
	StatusValue string

	// for logs
	// Log               string
}

type stateType int

const (
	invalid stateType = iota
	event
	statusCondition
	statusCustom
)

// NewWaitCondition constructs WaitCondition from provided configuration.
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
	wc.StatusKey, _ = waitOptions["status_key"].(string)
	wc.StatusValue, _ = waitOptions["status_value"].(string)

	wc.DeriveType()
	if !wc.Validate() {
		return nil, fmt.Errorf("format of condition for wait() is invalid; refer to documentation")
	}
	return
}

// DeriveType decides the type of WaitCondition.
func (wc *WaitCondition) DeriveType() {
	switch {
	case len(wc.Reason) > 0:
		wc.stateType = event
	case len(wc.ConditionType) > 0 && len(wc.Status) > 0:
		wc.stateType = statusCondition
	case len(wc.StatusKey) > 0 && len(wc.StatusValue) > 0:
		wc.stateType = statusCustom
	default:
		wc.stateType = invalid
	}
}

// Validate checks if WaitCondition makes sense.
func (wc *WaitCondition) Validate() bool {
	return wc.stateType > invalid &&
		len(wc.Kind) > 0 && len(wc.Namespace) > 0 && len(wc.Name) > 0
}

// TimeParams sets time parameters.
func (wc *WaitCondition) TimeParams(interval, timeout time.Duration) {
	if interval > 0 {
		wc.interval = interval
	}
	if timeout > 0 {
		wc.timeout = timeout
	}
}

// Build builds internal logic for WaitCondition.
func (wc *WaitCondition) Build() {
	switch wc.stateType {
	case statusCondition:
		wc.statusCondition()

	case statusCustom:
		wc.statusCustom()

	default: // == Event
		wc.event()
	}
}

func (wc *WaitCondition) statusCondition() {
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

			unstructured, err := c.dynamicClient.
				Resource(restMapping.Resource).
				Namespace(wc.Namespace).
				Get(ctx, wc.Name, metav1.GetOptions{})
			if err != nil {
				// From here on, we try to wait until resource reaches required state,
				// so don't return this error.
				// TODO add some logging here?
				return false, nil //nolint:nilerr
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

func (wc *WaitCondition) statusCustom() {
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

			unstructured, err := c.dynamicClient.
				Resource(restMapping.Resource).
				Namespace(wc.Namespace).
				Get(ctx, wc.Name, metav1.GetOptions{})
			if err != nil {
				// From here on, we try to wait until resource reaches required state,
				// so don't return this error.
				// TODO add some logging here?
				return false, nil //nolint:nilerr
			}

			v, found, err := metav1u.NestedString(unstructured.UnstructuredContent(), "status", wc.StatusKey)
			if err != nil {
				// conversion error should be returned
				return false, err
			}
			if !found {
				// Resource is without given status key: wait more in case
				// its .status changes.
				return false, nil
			}

			if v == wc.StatusValue {
				return true, nil
			}

			return false, nil
		}
	}
}

func (wc *WaitCondition) event() {
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
