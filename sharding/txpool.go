package sharding

import (
	"fmt"
)

// listenTXPool finds the pending tx's from the running geth node
// and sorts them by descending order of gas price, eliminates those
// that ask for too much gas, and routes them over to the VMC if the current
// node is a collator
func listenTXPool(c *Client) error {
	return fmt.Errorf("Not Implemented")
}
