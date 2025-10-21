package workflow

import "fmt"

// Ensure you update schema/workflow.schema.json on changes
type Dependent struct {
	_            struct{} `additionalProperties:"false" description:"A dependent configuration for external repositories"`
	Location     string   `yaml:"location" description:"The local path to the repository" required:"true"`
	CloneCommand string   `yaml:"cloneCommand,omitempty" description:"Optional command to clone the repository"`
}

func (r Dependent) Validate() error {
	if r.Location == "" {
		return fmt.Errorf("remote must have a location")
	}

	return nil
}
