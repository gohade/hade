package agent

import (
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/gohade/hade/framework/cobra"
)

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

func TestInitAgentCommandCanBeCalledTwiceWithoutStateLeak(t *testing.T) {
	first := InitAgentCommand()
	firstStart := findSubcommand(t, first, "start")
	if err := firstStart.Flags().Set("address", ":9999"); err != nil {
		t.Fatal(err)
	}
	if err := firstStart.Flags().Set("daemon", "true"); err != nil {
		t.Fatal(err)
	}

	second := InitAgentCommand()
	secondStart := findSubcommand(t, second, "start")
	if got := secondStart.Flags().Lookup("address").Value.String(); got != "" {
		t.Fatalf("second address leaked: %q", got)
	}
	if got := secondStart.Flags().Lookup("daemon").Value.String(); got != "false" {
		t.Fatalf("second daemon leaked: %q", got)
	}
}

func TestStartAgentServeReturnsListenerError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	server := &http.Server{Addr: listener.Addr().String()}
	err = startAgentServe(server, time.Second)
	if err == nil {
		t.Fatal("expected occupied address error")
	}
}

func TestListenFailureDoesNotWriteReady(t *testing.T) {
	readyFile := filepath.Join(t.TempDir(), "agent.ready")
	_, err := listenAndMarkReady(readyFile, 42, ":9999", func(string, string) (net.Listener, error) {
		return nil, errors.New("bind failed")
	})
	if err == nil {
		t.Fatal("expected bind error")
	}
	if _, err := os.Stat(readyFile); !os.IsNotExist(err) {
		t.Fatalf("ready file unexpectedly exists: %v", err)
	}
}

func TestDaemonAuthorizationEnvironmentDoesNotMutateParent(t *testing.T) {
	parent := []string{"A=1"}
	child := daemonChildEnvironment(parent, 4)
	if len(parent) != 1 || parent[0] != "A=1" {
		t.Fatalf("parent env mutated: %#v", parent)
	}
	if got := child[len(child)-1]; got != agentDaemonAuthFDEnv+"=4" {
		t.Fatalf("child auth fd = %q", got)
	}
}

func TestBuildDaemonArgsPreservesSpaceFormValues(t *testing.T) {
	options := newAgentOptions()
	options.address = "127.0.0.1:9999"
	*options.folderValues["runtime_folder"] = "/tmp/runtime folder"
	*options.folderValues["config_folder"] = "/tmp/config folder"

	got := buildDaemonArgs("hade", options)
	want := []string{
		"hade", "agent", "start", "--daemon=true",
		"--address", "127.0.0.1:9999",
		"--config_folder", "/tmp/config folder",
		"--runtime_folder", "/tmp/runtime folder",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func findSubcommand(t *testing.T, root interface{ Commands() []*cobra.Command }, name string) *cobra.Command {
	t.Helper()
	for _, command := range root.Commands() {
		if command.Name() == name {
			return command
		}
	}
	t.Fatalf("subcommand %q not found", name)
	return nil
}
