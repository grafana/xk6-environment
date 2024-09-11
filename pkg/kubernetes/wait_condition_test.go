package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_DeriveType(t *testing.T) {
	testCases := []struct {
		name         string
		wc           WaitCondition
		expectedType stateType
	}{
		{
			"empty is invalid",
			WaitCondition{state: state{}},
			invalid,
		},
		{
			"reason results in event",
			WaitCondition{state: state{Reason: "r"}},
			event,
		},
		{
			"status and value result in status condition type",
			WaitCondition{state: state{Status: "c", ConditionType: "s"}},
			statusCondition,
		},
		{
			"status key and value result in custom status type",
			WaitCondition{state: state{StatusKey: "k", StatusValue: "v"}},
			statusCustom,
		},
		{
			"status on its own is invalid",
			WaitCondition{state: state{Status: "s"}},
			invalid,
		},
		{
			"status key is invalid on its own",
			WaitCondition{state: state{StatusKey: "k"}},
			invalid,
		},
		{
			"condition on its own is invalid",
			WaitCondition{state: state{ConditionType: "c"}},
			invalid,
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.wc.DeriveType()
			assert.Equal(t, testCase.expectedType, testCase.wc.stateType)
		})
	}
}

func Test_Validate(t *testing.T) {
	testCases := []struct {
		name  string
		wc    WaitCondition
		valid bool
	}{
		{
			"empty is invalid",
			WaitCondition{},
			false,
		},
		{
			"invalid type",
			WaitCondition{resource: resource{Kind: "k", Namespace: "ns", Name: "n"}},
			false,
		},
		{
			"all valid for event type",
			WaitCondition{resource: resource{Kind: "k", Namespace: "ns", Name: "n"}, state: state{stateType: event}},
			true,
		},
		{
			"all valid for status condition type",
			WaitCondition{resource: resource{Kind: "k", Namespace: "ns", Name: "n"}, state: state{stateType: statusCondition}},
			true,
		},
		{
			"all valid for custom status type",
			WaitCondition{resource: resource{Kind: "k", Namespace: "ns", Name: "n"}, state: state{stateType: statusCustom}},
			true,
		},
		{
			"incomplete resource",
			WaitCondition{resource: resource{Kind: "k", Namespace: "ns"}, state: state{stateType: event}},
			false,
		},
		{
			"incomplete resource",
			WaitCondition{resource: resource{Kind: "k", Name: "n"}, state: state{stateType: event}},
			false,
		},
		{
			"incomplete resource",
			WaitCondition{resource: resource{Namespace: "ns", Name: "n"}, state: state{stateType: statusCondition}},
			false,
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.valid, testCase.wc.Validate())
		})
	}
}
