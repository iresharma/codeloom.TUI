# Product TUI

Go + Charm client for the Codeloom engine. Assumes the engine TUI protocol slice is already merged (`CreatePath`, `RenamePath`, `DeletePath`, `RequestContext`, `RequestMemory`, `RequestAgentTranscript`, full `OpenFile`, `MemoryUpdated`, `ContextBreakdown`, git-on-write).

`codeloom-tui` hosts the engine: attach if `{workspace}/.engine/engine.sock` already accepts connections, otherwise spawn `CODELOOM_ENGINE` → `engine` on `PATH` → sibling `engine` binary → `python3 app.py`. Owned children get `Shutdown` then SIGTERM/SIGKILL on quit. Attached engines are left running.

## Layout

```
+-- files (nvim) --+-- [chat] [file...] [agent...] --+-- stats / git ------+
| tree             |  default: orch chat             |  $  tokens  branch  |
| git glyphs       |  Enter on file -> new tab       |  staged/unstaged/?  |
| a A d r Enter    |  click agent -> transcript tab  +-- agent graph ------+
|                  |  file tab: full buffer,         |  orch               |
|                  |  ctrl+d toggles diff overlay    |    coder (live)     |
|                  |                                 +-- context | memory -+
|                  |                                 |  usage bar + dump   |
+------------------+---------------------------------+---------------------+
| > message or /command              [abort]                                |
```

## Keys

- `tab` / `ctrl+w` / `shift+tab` — cycle panes
- Tree: `hjkl`, `enter` open, `a` file, `A` dir, `r` rename, `d` delete, `R` refresh
- Center: `enter` send, `ctrl+t`/`ctrl+p` next/prev tab, `ctrl+e` close tab, `ctrl+d` file diff
- Right: `h`/`l` context|memory, `j`/`k` agents, `[`/`]` context sections, `enter` transcript, `r` refresh
- Global: `ctrl+x` abort, `ctrl+u` undo, `ctrl+c` quit
- Mouse: tabs, git paths, memory files, agent rows

## Stack

Bubble Tea + Bubbles + Lip Gloss + Glamour + BubbleZone. Protocol types live in `internal/protocol` and round-trip `testdata/protocol/golden.ndjson` copied from engine tests.

## Run

```
go run ./cmd/codeloom-tui [workspace]
```
