package tempo

type Plan struct {
	function any
	input    any
	name     string
}

func NewPlan(fn any, input any) Plan {
	return Plan{
		name:     getFunctionName(fn),
		function: fn,
		input:    input,
	}
}

func (p Plan) Name() string {
	return p.name
}

func (p Plan) Input() any {
	return p.input
}
