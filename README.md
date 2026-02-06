# go-libui

A minimal Go library for Plan 9 / 9front UI development.

## Overview

- **libui/** — Core library providing:
  - Disciplined UI control loop
  - Reducer-based state updates
  - Single redraw path
  - View-local state (scroll, geometry)

- **todo9/** — Example reactive todo list application demonstrating:
  - Scrolling and resizing
  - Widget-like behavior without widgets
  - No framework logic beyond app-local conventions

## Building

```sh
go build ./...
```

## Running (on Plan 9 / 9front)

```sh
./todo9
```

## Architecture

### libui

| File | Purpose |
|------|---------|
| `app.go` | Core types: `Event`, `Reducer`, `Drawer`, `App` |
| `event.go` | Raw input structs: `Mouse`, `Resize`, `Key` |
| `view.go` | View-local state: `ViewState` |
| `draw.go` | Thin wrapper around `/dev/draw` |
| `run.go` | Main event loop |

### todo9

| File | Purpose |
|------|---------|
| `model.go` | Application state: `Todo`, `Model` |
| `events.go` | Semantic events: `AddTodo`, `ToggleTodo`, etc. |
| `reducer.go` | Pure state transitions |
| `draw.go` | Rendering logic |
| `hit.go` | Manual hit-testing |
| `main.go` | Event translation and wiring |

## Constraints

- No external dependencies (pure Go + Plan 9 syscalls)
- No generics
- No reflection
- No goroutines except for input readers
- No global mutable state outside `ui.Run`
- Reducer never mutates view
- Drawer never mutates model

## Test Checklist

- [ ] Typing updates input line immediately
- [ ] Enter adds todo
- [ ] Clicking checkbox toggles state
- [ ] Scrolling does not affect model
- [ ] Resizing window recomputes layout
- [ ] No state mutation occurs in Draw
- [ ] No drawing occurs in Reduce
