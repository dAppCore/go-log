package main

import (
	"bytes"
	"testing"
)

func TestAX7_WriteCloser_Close_Good(t *testing.T) {
	var buf bytes.Buffer
	closer := nopWriteCloser{Writer: &buf}
	err := closer.Close()
	if err != nil {
		t.Fatalf("expected nil close error, got %v", err)
	}
}

func TestAX7_WriteCloser_Close_Bad(t *testing.T) {
	closer := nopWriteCloser{}
	err := closer.Close()
	if err != nil {
		t.Fatalf("zero-value closer should close without error, got %v", err)
	}
}

func TestAX7_WriteCloser_Close_Ugly(t *testing.T) {
	closer := nopWriteCloser{}
	first := closer.Close()
	second := closer.Close()
	if first != nil || second != nil {
		t.Fatalf("repeated close should be nil, first=%v second=%v", first, second)
	}
}
