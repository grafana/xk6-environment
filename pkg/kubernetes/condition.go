package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1u "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getConditions(ci []interface{}) []metav1.Condition {
	conditions := make([]metav1.Condition, 0)
	for _, c := range ci {
		cm, ok := c.(map[string]interface{})
		if ok {
			var cond metav1.Condition
			s, found, err := metav1u.NestedString(cm, "status")
			if err == nil && found {
				cond.Status = metav1.ConditionStatus(s)
			}

			t, found, err := metav1u.NestedString(cm, "type")
			if err == nil && found {
				cond.Type = t
			}
			conditions = append(conditions, cond)
		}
	}

	return conditions
}
