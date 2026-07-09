package slackcmd

type Deploy struct {
	Project string
	Env     string
}

func (d *Deploy) Name() string {
	return "Deploy"
}

type DeployBranchList struct {
	Project string
	Env     string
}

func (d *DeployBranchList) Name() string {
	return "DeployBranchList"
}

type DeployTargetSelection struct {
	Env string
}

func (d *DeployTargetSelection) Name() string {
	return "DeployTargetSelection"
}
