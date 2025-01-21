package tempo

import (
	"reflect"
	"runtime"
	"strings"
)

func getFunctionName(i any) (name string) {
	if fullName, ok := i.(string); ok {
		return fullName
	}

	fullName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()

	elements := strings.Split(fullName, ".")
	shortName := elements[len(elements)-1]

	return strings.TrimSuffix(shortName, "-fm")
}
