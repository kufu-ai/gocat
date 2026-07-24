package slackcmd

type Help struct{}

func (h *Help) Name() string {
	return "Help"
}

type ListProjects struct{}

func (l *ListProjects) Name() string {
	return "ListProjects"
}

type Reload struct{}

func (r *Reload) Name() string {
	return "Reload"
}
