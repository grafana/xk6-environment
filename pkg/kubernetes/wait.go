package kubernetes

import (
	"context"

	"k8s.io/apimachinery/pkg/util/wait"
)

// Wait blocks execution until wait condition is fulfilled.
func (c *Client) Wait(ctx context.Context, wc *WaitCondition) error {
	if err := wait.PollUntilContextTimeout(ctx, wc.interval, wc.timeout, true, wc.condF(c)); err != nil {
		return err
	}

	return nil
}
