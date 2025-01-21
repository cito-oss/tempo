package tempo

type Plan struct {
	name     string
	function any
	input    any
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
