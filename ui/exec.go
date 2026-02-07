package ui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/elizafairlady/go-libui/ui/proto"
	"github.com/elizafairlady/go-libui/ui/render"
	"github.com/elizafairlady/go-libui/ui/uifs"
	"github.com/elizafairlady/go-libui/ui/view"
)

// executor handles B2 command execution at the framework level.
// It checks builtins, then external commands.
type executor struct {
	app      view.App
	u        *uifs.UIFS
	r        *render.Renderer
	builtins map[string]view.Builtin
	binDirs  []string
}

// newExecutor creates an executor, extracting Executor info from the app
// if it implements the interface.
func newExecutor(app view.App, u *uifs.UIFS, r *render.Renderer) *executor {
	e := &executor{
		app: app,
		u:   u,
		r:   r,
	}
	if ex, ok := app.(view.Executor); ok {
		e.builtins = ex.Builtins()
		e.binDirs = ex.BinDirs()
	}
	return e
}

// execute handles a B2 execute action. Returns true if it handled
// the command (builtin or external), false if it should be passed
// to the app's Handle as a normal action.
func (e *executor) execute(act *proto.Action) bool {
	cmd := act.KVs["text"]
	id := act.KVs["id"]
	if cmd == "" {
		return false
	}

	// Build execution context
	ctx := &view.ExecContext{
		ID:    id,
		Cmd:   cmd,
		State: e.u.StateView(),
	}

	// Get selection from the focused body, if any
	if e.r != nil {
		ctx.Selection = e.r.BodySelection(e.u.Focus)
	}

	// 1. Check app builtins
	if e.builtins != nil {
		if builtin, ok := e.builtins[cmd]; ok {
			err := builtin(ctx)
			if err != nil {
				e.showError(id, cmd, err.Error())
			}
			return true
		}
	}

	// 2. Try external command
	path := e.findCommand(cmd)
	if path == "" {
		// Not found as builtin or external — fall through to app.Handle
		return false
	}

	// Run external command
	go e.runExternal(path, ctx)
	return true
}

// findCommand searches for cmd in the app's bin dirs and then PATH.
func (e *executor) findCommand(cmd string) string {
	// Don't try to run things with spaces or special chars as commands
	if strings.ContainsAny(cmd, " \t\n|&;(){}") {
		return ""
	}

	// Check app-specific bin dirs first
	for _, dir := range e.binDirs {
		path := dir + "/" + cmd
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}

	// Check PATH
	path, err := exec.LookPath(cmd)
	if err == nil {
		return path
	}

	return ""
}

// runExternal runs an external command with the execution context.
// stdin = selection, stdout → body/+Errors, env vars provide context.
func (e *executor) runExternal(path string, ctx *view.ExecContext) {
	cmd := exec.Command(path)

	// Environment variables matching Acme conventions
	cmd.Env = append(os.Environ(),
		"uiid="+ctx.ID,
		"uifocus="+e.u.Focus,
	)

	// stdin = current selection
	if ctx.Selection != "" {
		cmd.Stdin = strings.NewReader(ctx.Selection)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Collect output
	output := stdout.String()
	errOutput := stderr.String()

	if err != nil && errOutput == "" {
		errOutput = err.Error()
	}

	// If there's stdout, it could be inserted into the body
	// For now, just show any errors
	if errOutput != "" {
		e.showError(ctx.ID, ctx.Cmd, errOutput)
	}

	// If the command produced output, send it as a "cmdoutput" action
	// The app's Handle can decide what to do with it
	if output != "" {
		act := &proto.Action{
			Kind: "cmdoutput",
			KVs: map[string]string{
				"id":     ctx.ID,
				"cmd":    ctx.Cmd,
				"output": output,
			},
		}
		e.u.HandleAction(act)
	}

	_ = output
}

// showError sends an error to be displayed (typically in +Errors).
func (e *executor) showError(id, cmd, msg string) {
	act := &proto.Action{
		Kind: "cmderror",
		KVs: map[string]string{
			"id":    id,
			"cmd":   cmd,
			"error": fmt.Sprintf("%s: %s", cmd, msg),
		},
	}
	e.u.HandleAction(act)
}
