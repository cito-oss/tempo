package tasks

import (
	"context"
)

func PersonAge(ctx context.Context, name string) (int, error) {
	switch name {
	case "John Doe":
		return 35, nil

	case "Jane Doe":
		return 27, nil

	case "World":
		return 4.54e9, nil

	default:
		return 0, NewUnknownPersonError(name)
	}
}
