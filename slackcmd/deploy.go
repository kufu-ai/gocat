package slackcmd

type Deploy struct {
	Project string
	Env     string
}

func (d *Deploy) Name() string {
	return "Deploy"
}

func (d *Deploy) EnvName() string {
	return d.Env
}

type DeployBranchList struct {
	Project string
	Env     string
}

func (d *DeployBranchList) Name() string {
	return "DeployBranchList"
}

func (d *DeployBranchList) EnvName() string {
	return d.Env
}

type DeployTargetSelection struct {
	Env string
}

func (d *DeployTargetSelection) Name() string {
	return "DeployTargetSelection"
}

func (d *DeployTargetSelection) EnvName() string {
	return d.Env
}
