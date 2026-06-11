package article

import "testing"

func TestAIActionsIncludeWritingActions(t *testing.T) {
	for _, action := range []string{"metadata", "proofread", "polish", "expand", "shorten", "translate"} {
		t.Run(action, func(t *testing.T) {
			if _, ok := aiActions[action]; !ok {
				t.Fatalf("aiActions missing %q", action)
			}
		})
	}
}
