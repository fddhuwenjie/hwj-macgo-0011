package main

import (
	"testing"
)

func TestRunSelfCheck(t *testing.T) {
	if err := runSelfCheck(); err != nil {
		t.Fatal(err)
	}
}
