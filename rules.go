package main

import (
	"fmt"
	"regexp"
)

var (
	regex map[string]*regexp.Regexp
)

func init() {
	regex = make(map[string]*regexp.Regexp)
	regex["ticker"] = regexp.MustCompile(`^[\^A-Za-z0-9_\-:!\.#]+$`)
	regex["file"] = regexp.MustCompile(`^[A-Za-z0-9_\-\.]+$`)
}

func Validate(rule, input string) error {
	if _, ok := regex[rule]; !ok {
		return fmt.Errorf("No such rule=" + rule)
	}

	if !regex[rule].MatchString(input) {
		return fmt.Errorf("not matching %s (regexp=%s)", rule, regex[rule])
	}

	return nil
}
