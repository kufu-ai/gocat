package main

func NewInteractorKanavs(i InteractorContext) (o InteractorGitOps) {
	o = InteractorGitOps{
		InteractorContext: i,
		model:             NewGitOpsPluginKanvas(&o.github, &o.git),
	}
	o.kind = "kanvas"
	return
}
