package store

type TargetGroup struct {
	Name    string            `json:"-"`
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

func (ts *TargetGroup) SetLabels(labels map[string]string) {
	ts.Labels = labels
}
