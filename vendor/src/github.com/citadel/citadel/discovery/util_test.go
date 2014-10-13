package discovery

import "testing"

var local = "http://localhost:8080"

func TestRegisterLocal(t *testing.T) {
	expected := "127.0.0.1:2675"
	if err := RegisterSlave(local, "crosbymichael", "test", "node1", expected); err != nil {
		t.Fatal(err)
	}

	addrs, err := FetchSlaves(local, "crosbymichael", "test")
	if err != nil {
		t.Fatal(err)
	}

	if len(addrs) == 0 {
		t.Fatal("expected addr len == 1")
	}

	if addrs[0] != expected {
		t.Fatalf("expected addr %q but received %q", expected, addrs[0])
	}
}
