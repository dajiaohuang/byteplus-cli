package cmd

import (
	"strings"
	"testing"
)

func TestUsageTemplatesIncludeFixedFlags(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "root", text: rootUsageTemplate()},
		{name: "service", text: serviceUsageTemplate()},
		{name: "action", text: actionUsageTemplate("", []string{"InstanceId string"})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, want := range expectedFixedFlagsForTest() {
				if !strings.Contains(tt.text, want) {
					t.Fatalf("%s usage missing %q:\n%s", tt.name, want, tt.text)
				}
			}
		})
	}
}

func TestJSONActionUsageSeparatesParameterForms(t *testing.T) {
	want := "Available Parameters:\n\n" +
		"  Parameter Form:\n" +
		"    --Filter.Name string\n" +
		"    --PageSize integer\n\n" +
		"  JSON Form:\n" +
		"    --body '{\n" +
		"        \"Filter\": {}\n" +
		"    }'"

	out := jsonActionUsageTemplate("", []string{"PageSize integer", "Filter.Name string"}, "body '{\n    \"Filter\": {}\n}'")
	if !strings.Contains(out, want) {
		t.Fatalf("JSON action usage missing grouped parameters:\n%s", out)
	}
}

func TestNonJSONActionUsageKeepsSingleParameterList(t *testing.T) {
	out := actionUsageTemplate("", []string{"InstanceId string"})
	if !strings.Contains(out, "Available Parameters:\n  --InstanceId string") {
		t.Fatalf("non-JSON action usage changed unexpectedly:\n%s", out)
	}
	for _, unwanted := range []string{"Parameter Form:", "JSON Form:"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("non-JSON action usage unexpectedly contains %q:\n%s", unwanted, out)
		}
	}
}

func TestJSONActionUsageOmitsEmptyParameterForm(t *testing.T) {
	out := jsonActionUsageTemplate("", nil, "body '{}'")
	if strings.Contains(out, "Parameter Form:") {
		t.Fatalf("JSON action usage contains an empty parameter form:\n%s", out)
	}
	if !strings.Contains(out, "JSON Form:\n    --body '{}'") {
		t.Fatalf("JSON action usage missing body form:\n%s", out)
	}
}

func expectedFixedFlagsForTest() []string {
	return []string{"---profile", "---region", "---endpoint"}
}
