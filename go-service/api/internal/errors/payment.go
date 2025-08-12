package errors

import "errors"

var ErrNoPaymentProcessorAvailable = errors.New("no processor is available")
var ErrNoPaymentsInQueue = errors.New("no payments in queue")
