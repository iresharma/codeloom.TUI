package protocol

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEncodeStartSession(t *testing.T) {
	raw, err := Encode(StartSession("/tmp/ws", ""))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(bytesTrim(raw), &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "StartSession" || m["workspace"] != "/tmp/ws" {
		t.Fatalf("got %#v", m)
	}
}

func TestDecodeEvent(t *testing.T) {
	ev, err := DecodeEvent([]byte(`{"type":"FileContent","path":"a.py","content":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != "FileContent" || ev.Str("path") != "a.py" {
		t.Fatalf("got %#v", ev)
	}
}

func TestGoldenNDJSON(t *testing.T) {
	path := goldenPath(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantTypes := map[string]bool{
		"CreatePath": true, "RenamePath": true, "DeletePath": true,
		"RequestContext": true, "RequestMemory": true, "RequestAgentTranscript": true,
		"RequestSnapshot": true, "OpenFile": true, "PathChanged": true,
		"ContextBreakdown": true, "MemoryUpdated": true, "GitStateUpdated": true,
		"FileContent": true,
	}
	seen := map[string]bool{}
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		ev, err := DecodeEvent(line)
		if err != nil {
			t.Fatalf("decode %s: %v", line, err)
		}
		seen[ev.Type] = true
		if !wantTypes[ev.Type] {
			t.Fatalf("unexpected type %s", ev.Type)
		}
	}
	for typ := range wantTypes {
		if !seen[typ] {
			t.Fatalf("missing golden type %s", typ)
		}
	}
}

func TestConstructorsMatchGolden(t *testing.T) {
	raw, err := Encode(CreatePath("a.py", false, "x"))
	if err != nil {
		t.Fatal(err)
	}
	ev, err := DecodeEvent(bytesTrim(raw))
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != "CreatePath" || ev.Str("path") != "a.py" || ev.Str("content") != "x" || ev.Bool("is_dir") {
		t.Fatalf("got %#v", ev.Raw)
	}
	snap, err := Encode(RequestSnapshot(true))
	if err != nil {
		t.Fatal(err)
	}
	sev, err := DecodeEvent(bytesTrim(snap))
	if err != nil {
		t.Fatal(err)
	}
	if sev.Type != "RequestSnapshot" || !sev.Bool("replay") {
		t.Fatalf("got %#v", sev.Raw)
	}
}

func TestContextAndMemoryShape(t *testing.T) {
	ev, err := DecodeEvent([]byte(`{"type":"ContextBreakdown","agent_id":"","budget":120000,"prompt_tokens":0,"compacted":false,"sections":[{"name":"system","chars":5,"tokens_est":1,"text":"hello"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	secs := ev.Slice("sections")
	if len(secs) != 1 {
		t.Fatalf("sections %v", secs)
	}
	row := secs[0].(map[string]any)
	if row["name"] != "system" || row["tokens_est"].(float64) != 1 {
		t.Fatalf("section %#v", row)
	}
}

func goldenPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "protocol", "golden.ndjson")
}

func bytesTrim(b []byte) []byte {
	return bytes.TrimSpace(b)
}
