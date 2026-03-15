package config

import "testing"

func TestPersistentEditsUsesPatchFiles(t *testing.T) {
	t.Parallel()

	trueValue := true
	falseValue := false

	tests := []struct {
		name string
		cfg  *PersistentEdits
		want bool
	}{
		{name: "nil", cfg: nil, want: false},
		{name: "missing", cfg: &PersistentEdits{}, want: false},
		{name: "true", cfg: &PersistentEdits{PatchFiles: &trueValue}, want: true},
		{name: "false", cfg: &PersistentEdits{PatchFiles: &falseValue}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.cfg.UsesPatchFiles(); got != tt.want {
				t.Fatalf("UsesPatchFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}
