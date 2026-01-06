package tool

type Tool interface{}

type tool struct{}

func New() Tool {
	return &tool{}
}
