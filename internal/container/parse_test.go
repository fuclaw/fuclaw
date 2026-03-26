package container

import (
	"testing"
)

func TestParseOutputLine_NoMarker(t *testing.T) {
	out, matched, err := ParseOutputLine("hello")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if matched {
		t.Fatalf("expected matched=false")
	}
	if out != nil {
		t.Fatalf("expected out=nil")
	}
}

func TestParseOutputLine_Valid(t *testing.T) {
	line := OutputStartMarker + `{"status":"success","result":"ok"}` + OutputEndMarker
	out, matched, err := ParseOutputLine(line)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if !matched {
		t.Fatalf("expected matched=true")
	}
	if out == nil {
		t.Fatalf("expected out!=nil")
	}
	if out.Status != "success" {
		t.Fatalf("unexpected status: %s", out.Status)
	}
}

func TestParseOutputLine_InvalidJSON(t *testing.T) {
	line := OutputStartMarker + `{"status":` + OutputEndMarker
	_, matched, err := ParseOutputLine(line)
	if !matched {
		t.Fatalf("expected matched=true")
	}
	if err == nil {
		t.Fatalf("expected err!=nil")
	}
}

func TestParseOutputLine_MissingEndMarker(t *testing.T) {
	line := OutputStartMarker + `{"status":"success"}`
	_, matched, err := ParseOutputLine(line)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if matched {
		t.Fatalf("expected matched=false when missing end marker")
	}
}

