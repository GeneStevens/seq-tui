package main

import (
	"strings"
	"testing"
)

func TestBannerIsOneLine(t *testing.T) {
	if Banner == "" || strings.Contains(Banner, "\n") {
		t.Fatalf("banner must be a non-empty single line, got %q", Banner)
	}
}
