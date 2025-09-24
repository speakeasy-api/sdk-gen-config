package workflow

import "fmt"

// Ensure you update schema/workflow.schema.json on changes
type Remote struct {
	GithubRepo string   `yaml:"github_repo,omitempty"`
	Location   string   `yaml:"location"`
	Groups     []string `yaml:"groups,omitempty"` // Used to enable dispatching workflow runs to subsets ("groups") of remotes
}

func (r Remote) Validate() error {
	if r.Location == "" {
		return fmt.Errorf("remote must have a location")
	}

	return nil
}
