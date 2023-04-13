package errors

import "errors"

var (
	ErrUserExists           = errors.New("user exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrOrderExists          = errors.New("order exists")
	ErrOrderNotBelongToUser = errors.New("order does not belong to user")
	ErrInsufficientFunds    = errors.New("insufficient funds")
)
