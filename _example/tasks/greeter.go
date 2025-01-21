package tasks

import (
	"context"
)

func Greeter(ctx context.Context, name string) (string, error) {
	if name == "" {
		name = "World"
	}

	return "Hello " + name, nil
}
