package main

import (
	"testing"
)

func TestFiles(t *testing.T) {
	for ticker, valid := range map[string]bool{
		"/tmp/file.txt": false,
		"../file.txt":   false,
		"file.txt":      true,
		"add-ticker.sh": true,
	} {
		e := Validate("file", ticker)
		if valid && e != nil {
			t.Errorf("Ticker(%s) should not fail e=%s\n", ticker, e.Error())
		}
		if !valid && e == nil {
			t.Errorf("Ticker(%s) should fail e=%s\n", ticker, e.Error())
		}
	}
}
