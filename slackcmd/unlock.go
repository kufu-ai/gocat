package slackcmd

type Unlock struct {
	Project string
	Env     string
}

func (u *Unlock) Name() string {
	return "Unlock"
}

func (u *Unlock) EnvName() string {
	return u.Env
}
