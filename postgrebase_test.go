package postgrebase

import "testing"

func TestFilterEagerFlagArgsSkipsSubcommandFlags(t *testing.T) {
	knownFlags := map[string]bool{
		"dataDsn":   true,
		"peers":     true,
		"node-addr": true,
		"node-id":   true,
	}
	boolFlags := map[string]bool{}

	args := []string{
		"serve",
		"--http", "127.0.0.1:8090",
		"--node-addr", "http://127.0.0.1:8090",
		"--peers=http://127.0.0.1:8091",
		"--node-id", "node-a",
	}

	got := filterEagerFlagArgs(args, knownFlags, boolFlags)
	want := []string{
		"--node-addr", "http://127.0.0.1:8090",
		"--peers=http://127.0.0.1:8091",
		"--node-id", "node-a",
	}

	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestInferNodeAddrFromHTTPArg(t *testing.T) {
	got := inferNodeAddrFromArgs([]string{
		"serve",
		"--http", "127.0.0.1:8091",
		"--peers", "http://127.0.0.1:8090",
	})

	if got != "http://127.0.0.1:8091" {
		t.Fatalf("expected inferred node addr, got %q", got)
	}
}

func TestNodeAddrFromHTTPAddrNormalizesWildcardHost(t *testing.T) {
	got := nodeAddrFromHTTPAddr("0.0.0.0:8090")

	if got != "http://127.0.0.1:8090" {
		t.Fatalf("expected localhost node addr, got %q", got)
	}
}
