package kubernetes

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

func (c *Client) Wait(ctx context.Context, wc *WaitCondition) error {
	if wc.isTestExecution {
		// TODO: hard cap of 1h for any test here
		if err := wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Hour, true, wc.condF); err != nil {
			return err
		}
	}

	if wc.isTimeout {
		time.Sleep(wc.duration)
	}

	return nil
}
