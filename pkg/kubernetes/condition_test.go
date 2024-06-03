package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_DeriveType(t *testing.T) {
	testCases := []struct {
		name         string
		wc           WaitCondition
		expectedType StateType
	}{
		{
			"empty is invalid",
			WaitCondition{State: State{}},
			Invalid,
		},
		{
			"reason results in event",
			WaitCondition{State: State{Reason: "r"}},
			Event,
		},
		{
			"status and value result in status type",
			WaitCondition{State: State{Value: "c", Status: "s"}},
			Status,
		},
		{
			"status on its own is invalid",
			WaitCondition{State: State{Status: "s"}},
			Invalid,
		},
		{
			"condition results in status type",
			WaitCondition{State: State{Condition: "c"}},
			Status,
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			testCase.wc.DeriveType()
			assert.Equal(t, testCase.expectedType, testCase.wc.StateType)
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
			WaitCondition{Resource: Resource{Kind: "k", Namespace: "ns", Name: "n"}},
			false,
		},
		{
			"all valid for event type",
			WaitCondition{Resource: Resource{Kind: "k", Namespace: "ns", Name: "n"}, State: State{StateType: Event}},
			true,
		},
		{
			"all valid for status type",
			WaitCondition{Resource: Resource{Kind: "k", Namespace: "ns", Name: "n"}, State: State{StateType: Status}},
			true,
		},
		{
			"incomplete resource",
			WaitCondition{Resource: Resource{Kind: "k", Namespace: "ns"}, State: State{StateType: Event}},
			false,
		},
		{
			"incomplete resource",
			WaitCondition{Resource: Resource{Kind: "k", Name: "n"}, State: State{StateType: Event}},
			false,
		},
		{
			"incomplete resource",
			WaitCondition{Resource: Resource{Namespace: "ns", Name: "n"}, State: State{StateType: Status}},
			false,
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.valid, testCase.wc.Validate())
		})
	}
}
