package network

import "errors"

var ErrorConnectionClosed = errors.New("Connection is closed")
var ErrorMessageTooLarge = errors.New("Message is too large!")
