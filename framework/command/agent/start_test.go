package agent

import "testing"

func TestResolveAgentAddressPriority(t *testing.T) {
	tests := []struct {
		name         string
		flagAddress  string
		envAddress   string
		configPort   string
		configExists bool
		want         string
	}{
		{name: "flag first", flagAddress: "127.0.0.1:9001", envAddress: "127.0.0.1:9002", configPort: "9003", configExists: true, want: "127.0.0.1:9001"},
		{name: "env second", envAddress: "127.0.0.1:9002", configPort: "9003", configExists: true, want: "127.0.0.1:9002"},
		{name: "config third", configPort: "9003", configExists: true, want: ":9003"},
		{name: "default", want: ":8889"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveAgentAddress(tt.flagAddress, tt.envAddress, tt.configPort, tt.configExists)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeAgentAddress(t *testing.T) {
	for input, want := range map[string]string{
		"8889":  ":8889",
		":8889": ":8889",
		"":      ":8889",
	} {
		if got := normalizeAgentAddress(input); got != want {
			t.Fatalf("normalizeAgentAddress(%q) = %q, want %q", input, got, want)
		}
	}
}
