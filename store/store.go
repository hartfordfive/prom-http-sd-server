package store

type DataStore interface {
	AddTargetToGroup(targetGroup, target string) error
	RemoveTargetFromGroup(targetGroup, target string) error
	GetTargetGroupLabels(targetGroup string) (*map[string]string, error)
	AddLabelsToGroup(targetGroup string, labels map[string]string) error
	RemoveLabelFromGroup(targetGroup, label string) error
	Serialize(debug bool) (string, error)
	Shutdown()
}

var StoreInstance *DataStore
