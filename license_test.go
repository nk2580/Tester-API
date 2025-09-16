package main

import (
	"os"
	"strings"
	"testing"
)

func TestLicenseFileExists(t *testing.T) {
	data, err := os.ReadFile("LICENSE")
	if err != nil {
		t.Fatalf("LICENSE file missing: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Permission is hereby granted, free of charge") {
		t.Fatalf("LICENSE does not contain canonical MIT phrase")
	}
	if !strings.Contains(content, "SPDX-License-Identifier: MIT") && !strings.Contains(content, "MIT License") {
		t.Fatalf("LICENSE missing SPDX identifier or 'MIT License' header")
	}
}

func TestReadmeLinksLicense(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md missing: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "[LICENSE](./LICENSE)") && !strings.Contains(content, "[License](./LICENSE)") {
		t.Fatalf("README.md does not link to ./LICENSE")
	}
}
