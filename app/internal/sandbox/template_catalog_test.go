package sandbox

import "testing"

func TestIsStandardTemplateRecognizesProviderScopedName(t *testing.T) {
	for _, name := range []string{"gaze", "team/gaze", "project-b89e/Gaze"} {
		if !isStandardTemplate(name) {
			t.Fatalf("expected %q to identify the Gaze standard template", name)
		}
	}
	if isStandardTemplate("gaze-custom") {
		t.Fatal("custom template must not be treated as the standard template")
	}
}
