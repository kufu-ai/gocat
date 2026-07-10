package slackcmd

type Command interface {
	Name() string
}

type EnvCommand interface {
	Command
	EnvName() string
}
