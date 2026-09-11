package protocol

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const StreamLimit = 8 * 1024 * 1024

type Command map[string]any

func Cmd(typ string, fields map[string]any) Command {
	out := Command{"type": typ}
	for k, v := range fields {
		if v == nil {
			continue
		}
		out[k] = v
	}
	return out
}

func StartSession(workspace, sessionID string) Command {
	fields := map[string]any{"workspace": workspace}
	if sessionID != "" {
		fields["session_id"] = sessionID
	}
	return Cmd("StartSession", fields)
}

func SubmitUserMessage(text string) Command {
	return Cmd("SubmitUserMessage", map[string]any{"text": text})
}

func OpenFile(path string) Command { return Cmd("OpenFile", map[string]any{"path": path}) }
func CloseFile(path string) Command {
	return Cmd("CloseFile", map[string]any{"path": path})
}
func CreatePath(path string, isDir bool, content string) Command {
	return Cmd("CreatePath", map[string]any{"path": path, "is_dir": isDir, "content": content})
}
func RenamePath(src, dest string) Command {
	return Cmd("RenamePath", map[string]any{"src": src, "dest": dest})
}
func DeletePath(path string) Command {
	return Cmd("DeletePath", map[string]any{"path": path})
}
func RequestSnapshot(replay bool) Command {
	return Cmd("RequestSnapshot", map[string]any{"replay": replay})
}
func RequestGit() Command    { return Cmd("RequestGit", nil) }
func RequestMemory() Command { return Cmd("RequestMemory", nil) }
func RequestContext(id string) Command {
	return Cmd("RequestContext", map[string]any{"agent_id": id})
}
func RequestAgentTranscript(id string) Command {
	return Cmd("RequestAgentTranscript", map[string]any{"agent_id": id})
}
func AbortAgent(id string) Command {
	fields := map[string]any{}
	if id != "" {
		fields["agent_id"] = id
	}
	return Cmd("AbortAgent", fields)
}
func AnswerPrompt(id, text string) Command {
	return Cmd("AnswerPrompt", map[string]any{"prompt_id": id, "text": text})
}
func Shutdown() Command { return Cmd("Shutdown", nil) }
func UndoLastEdit() Command {
	return Cmd("UndoLastEdit", nil)
}

func Encode(cmd Command) ([]byte, error) {
	raw, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

type Event struct {
	Type string
	Raw  map[string]any
}

func DecodeEvent(line []byte) (Event, error) {
	line = bytes.TrimSpace(line)
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err != nil {
		return Event{}, err
	}
	typ, _ := raw["type"].(string)
	if typ == "" {
		return Event{}, fmt.Errorf("event missing type")
	}
	return Event{Type: typ, Raw: raw}, nil
}

func (e Event) Str(key string) string {
	v, _ := e.Raw[key].(string)
	return v
}

func (e Event) Bool(key string) bool {
	v, _ := e.Raw[key].(bool)
	return v
}

func (e Event) Int(key string) int {
	switch v := e.Raw[key].(type) {
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	}
	return 0
}

func (e Event) Float(key string) float64 {
	switch v := e.Raw[key].(type) {
	case float64:
		return v
	case json.Number:
		n, _ := v.Float64()
		return n
	}
	return 0
}

func (e Event) Map(key string) map[string]any {
	v, _ := e.Raw[key].(map[string]any)
	if v == nil {
		return map[string]any{}
	}
	return v
}

func (e Event) Slice(key string) []any {
	v, _ := e.Raw[key].([]any)
	return v
}

func NewScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, StreamLimit)
	return sc
}

func AsStringSlice(v any) []string {
	arr, _ := v.([]any)
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, _ := item.(string)
		out = append(out, s)
	}
	return out
}
