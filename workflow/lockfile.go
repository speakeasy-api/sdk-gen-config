package workflow

type LockFile struct {
	SpeakeasyVersion string                `yaml:"speakeasyVersion"`
	Sources          map[string]SourceLock `yaml:"sources"`
	Targets          map[string]TargetLock `yaml:"targets"`

	Workflow Workflow `yaml:"workflow"`
}

type SourceLock struct {
	SourceRevisionDigest string   `yaml:"sourceRevisionDigest,omitempty"`
	SourceNamespaceName  string   `yaml:"sourceNamespaceName,omitempty"`
	Tags                 []string `yaml:"tags,omitempty"`
}

type TargetLock struct {
	Source          string `yaml:"source,omitempty"`
	GenLockLocation string `yaml:"genLockLocation,omitempty"`
}
