package coredata

import (
	"os"
	"testing"
)

const testPath = "/tmp/coredata"

func TestNewDatabase(t *testing.T) {
	d, err := New(testPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbdefer(d)

	if d == nil {
		t.Fatal("database should not be nil")
	}
}

func TestCreateGroup(t *testing.T) {
	d, err := New(testPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbdefer(d)

	name := "test-group"

	if err := d.CreateGroup(name); err != nil {
		t.Fatal(err)
	}

	groups, err := d.ListGroups()
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) != 1 {
		t.Fatalf("expected group length of 1 but received %d", len(groups))
	}

	actual := groups[0]

	if actual != name {
		t.Fatalf("expected name %q but received %q", name, actual)
	}

	if err := d.RemoveGroup(name); err != nil {
		t.Fatal(err)
	}

	if groups, err = d.ListGroups(); err != nil {
		t.Fatal(err)
	}

	if len(groups) != 0 {
		t.Fatalf("expected group length 0 but received %d", len(groups))
	}
}

func TestCreateContainer(t *testing.T) {
	d, err := New(testPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbdefer(d)

	name := "test-group"

	if err := d.CreateGroup(name); err != nil {
		t.Fatal(err)
	}

	conatinerid := "1"

	if err := d.AddContainerToGroup(name, conatinerid); err != nil {
		t.Fatal(err)
	}

	ids, err := d.ListContainersInGroup(name)
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 1 {
		t.Fatalf("expected containers length of 1 but received %d", len(ids))
	}

	actual := ids[0]

	if actual != conatinerid {
		t.Fatalf("expected container id %q but received %q", conatinerid, actual)
	}

	if err := d.RemoveContainerFromGroup(name, conatinerid); err != nil {
		t.Fatal(err)
	}

	if ids, err = d.ListContainersInGroup(name); err != nil {
		t.Fatal(err)
	}

	if len(ids) != 0 {
		t.Fatalf("expected containers length of 0 but received %d", len(ids))
	}
}

func dbdefer(d *Coredata) {
	d.Close()
	os.RemoveAll(testPath)
}
