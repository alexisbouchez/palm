package agent

type Agent interface {
	Run() error
}

type agent struct{}

func New() Agent {
	return &agent{}
}

func (a *agent) Run() error {
	return nil
}
