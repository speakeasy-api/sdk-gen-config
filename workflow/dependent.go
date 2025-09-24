package workflow

import "fmt"

// Ensure you update schema/workflow.schema.json on changes
type Dependent struct {
	Location     string `yaml:"location"`
	CloneCommand string `yaml:"cloneCommand,omitempty"`
}

func (r Dependent) Validate() error {
	if r.Location == "" {
		return fmt.Errorf("remote must have a location")
	}

	return nil
}
