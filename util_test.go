package main

import (
	"testing"
)

func TestSliceContainsStr(t *testing.T) {
	for _, tt := range []struct {
		s []string
		b string
		r bool
	}{
		{[]string{}, "one", false},
		{[]string{}, "", false},
		{[]string{"one"}, "one", true},
		{[]string{"one"}, "", false},
		{[]string{"one"}, "two", false},
		{[]string{"one", "two"}, "one", true},
		{[]string{"one", "two"}, "three", false},
	} {
		r := sliceContainsStr(tt.s, tt.b)
		if r != tt.r {
			t.Errorf("sliceContainsStr(%v, %s) => %t, want %t", tt.s, tt.b, r, tt.r)
		}
	}
}

func TestSliceEqStr(t *testing.T) {
	for _, tt := range []struct {
		a []string
		b []string
		r bool
	}{
		{[]string{}, []string{}, true},
		{[]string{"one"}, []string{}, false},
		{[]string{"one"}, []string{"one"}, true},
		{[]string{"one"}, []string{"two"}, false},
		{[]string{"one"}, []string{"one", "two"}, false},
	} {
		r := sliceEqStr(tt.a, tt.b)
		if r != tt.r {
			t.Errorf("sliceEqStr(%v, %v) => %t, want %t", tt.a, tt.b, r, tt.r)
		}
	}
}

func TestSliceSubStr(t *testing.T) {
	for _, tt := range []struct {
		a []string
		b []string
		r []string
	}{
		{[]string{}, []string{}, []string{}},
		{[]string{}, []string{"one"}, []string{}},
		{[]string{}, []string{""}, []string{}},
		{[]string{"one"}, []string{"one"}, []string{}},
		{[]string{"one"}, []string{""}, []string{"one"}},
		{[]string{"one"}, []string{"two"}, []string{"one"}},
		{[]string{"one", "two"}, []string{"one"}, []string{"two"}},
		{[]string{"one", "two"}, []string{"three"}, []string{"one", "two"}},
	} {
		r := sliceSubStr(tt.a, tt.b)
		if !sliceEqStr(r, tt.r) {
			t.Errorf("sliceSubStr(%v, %v) => %v, want %v", tt.a, tt.b, r, tt.r)
		}
	}
}
