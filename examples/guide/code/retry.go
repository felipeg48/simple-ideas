//go:build ignore

package payments

import (
	"context"
	"errors"
	"time"
)

// Pay authorises a card payment, retrying once when the bank times out.
func Pay(ctx context.Context, bank Bank, o Order, card Card, amount Money) (Result, error) {
	// tag::retry[]
	key := o.ID                                           // <1>
	res, err := bank.Authorize(ctx, card, amount, key)
	if errors.Is(err, ErrTimeout) {
		time.Sleep(30 * time.Second)                      // <2>
		res, err = bank.Authorize(ctx, card, amount, key)
	}
	return res, err                                       // <3>
	// end::retry[]
}
