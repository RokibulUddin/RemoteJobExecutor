package remotejob

import (
	"fmt"
	"strings"
	"testing"
)

// TestNewHostFromRecord test NewHostFromRecord
func TestNewHostFromRecord(t *testing.T) {
	str := "NAME, 127.0.0.1, root, pass, ls -al, hola"
	testHost, err := NewHostFromRecord(strings.Split(str, ","))
	fmt.Println(testHost)
	if err != nil {
		t.Error(err)
	}
	if testHost.Name != "NAME" {
		t.Error("Name parsing error")
	}
	if testHost.IP != "127.0.0.1" {
		t.Error("IP parsing error")
	}
	if testHost.User != "root" {
		t.Error("User parsing error")
	}
	if testHost.Password != "pass" {
		t.Error("Password parsing error")
	}
	if testHost.CMD != "ls -al, hola" {
		t.Error("Command parsing error!")
	}
}
