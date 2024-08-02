package slackcmd

type Unlock struct {
	Project string
	Env     string
}

func (u *Unlock) Name() string {
	return "Unlock"
}
