package tempo

type Test interface {
	Name() string
	Function() any
}

// NewTest wraps fn in a workflowWrapper, that can be passed to temporal Worker:
//
//	mytest := tempo.NewTest(func(*T))
//
//	w.RegisterWorkflowWithOptions(mytest.Function(), workflow.RegisterOptions{
//		Name: mytest.Name(),
//	})
func NewTest(fn func(*T)) Test {
	return &workflowWrapper[any, any]{
		name: getFunctionName(fn),
		fn:   fn,
	}
}

// NewTestWithInput wraps fn in a workflowWrapper, that can be passed to temporal Worker:
//
//	mytest := tempo.NewTestWithInput(func(*T, string))
//
//	w.RegisterWorkflowWithOptions(mytest.Function(), workflow.RegisterOptions{
//		Name: mytest.Name(),
//	})
func NewTestWithInput[INPUT any](fn func(*T, INPUT)) Test {
	return &workflowWrapper[INPUT, any]{
		name:     getFunctionName(fn),
		fnWithIn: fn,
	}
}

// NewTestWithOutput wraps fn in a workflowWrapper, that can be passed to temporal Worker:
//
//	mytest := tempo.NewTestWithOutput(func(*T) string)
//
//	w.RegisterWorkflowWithOptions(mytest.Function(), workflow.RegisterOptions{
//		Name: mytest.Name(),
//	})
func NewTestWithOutput[OUTPUT any](fn func(*T) OUTPUT) Test {
	return &workflowWrapper[any, OUTPUT]{
		name:      getFunctionName(fn),
		fnWithOut: fn,
	}
}

// NewTestWithInputAndOutput wraps fn in a workflowWrapper, that can be passed to temporal Worker:
//
//	mytest := tempo.NewTestWithInputAndOutput(func(*T, string) string)
//
//	w.RegisterWorkflowWithOptions(mytest.Function(), workflow.RegisterOptions{
//		Name: mytest.Name(),
//	})
func NewTestWithInputAndOutput[INPUT any, OUTPUT any](fn func(*T, INPUT) OUTPUT) Test {
	return &workflowWrapper[INPUT, OUTPUT]{
		name:           getFunctionName(fn),
		fnWithInAndOut: fn,
	}
}
