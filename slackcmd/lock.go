package slackcmd

type Lock struct {
	Project string
	Env     string
	Reason  string
}

func (l *Lock) Name() string {
	return "Lock"
}
