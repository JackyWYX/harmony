package requestmanager

import "time"

// TODO: determine the values in production environment
const (
	// throttle to do request every 100 milliseconds
	throttleInterval = 100 * time.Millisecond

	// number of request to be done in each throttle loop
	throttleBatch = 16

	// default module built-in time out for a request
	reqTimeOut = 5 * time.Second

	// deliverTimeout is the timeout for a response delivery.
	deliverTimeout = 1 * time.Minute

	// maxWaitingSize is the maximum requests that are in waiting list
	maxWaitingSize = 1024
)
