package internal

import (
	"errors"
	"log/slog"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/atlasopsai-star/Orbit/src/pkg/utils"
)

type IgnorerWriter struct{}

func (w IgnorerWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

type TeaProg struct {
	m    *model
	prog *tea.Program
}

func NewTestTeaProgWithEventLoop(t *testing.T, m *model) *TeaProg {
	p := &TeaProg{m: m, prog: tea.NewProgram(m,
		tea.WithInput(nil), tea.WithOutput(IgnorerWriter{}),
		tea.WithWindowSize(DefaultTestModelWidth, DefaultTestModelHeight))}
	p.StartEventLoop()

	t.Cleanup(func() {
		p.Close()
	})
	return p
}

func NewTestTeaProgWithEventLoopWithWinSize(t *testing.T, m *model, width int, height int) *TeaProg {
	p := &TeaProg{m: m, prog: tea.NewProgram(m,
		tea.WithInput(nil), tea.WithOutput(IgnorerWriter{}),
		tea.WithWindowSize(width, height))}
	p.StartEventLoop()
	t.Cleanup(func() {
		p.Close()
	})
	return p
}

func (p *TeaProg) getModel() *model {
	return p.m
}

func (p *TeaProg) StartEventLoop() {
	go func() {
		_, err := p.prog.Run()
		// This will return only after Run() is completed
		// This will not be error if its due to p.Close() being called
		if err != nil && !errors.Is(err, tea.ErrProgramKilled) {
			slog.Error("TeaProg finished with error", "error", err)
		}
	}()
	// Send nil to block for start of event loop
	p.prog.Send(nil)
}

func (p *TeaProg) Send(msgs ...tea.Msg) {
	for _, msg := range msgs {
		p.prog.Send(msg)
	}
}

func (p *TeaProg) SendKey(key string) {
	p.Send(utils.TeaRuneKeyMsg(key))
}

// NavigateToDir changes the focused panel's directory through the event loop
// and blocks until the panel has landed on the target with its elements loaded.
// The navigation message and the follow-up nil update are applied via
// SendDirectly so element loading happens synchronously on this goroutine,
// serialized against the running event loop by the model mutex.
func (p *TeaProg) NavigateToDir(t *testing.T, dir string) {
	t.Helper()
	p.SendDirectly(navigateToDirMsg{dir: dir})
	p.SendDirectly(nil)
	require.Eventually(t, func() bool {
		return p.getModel().getFocusedFilePanel().GetLocation() == dir
	}, DefaultTestTimeout, DefaultTestTick, "Focused panel should navigate to %s", dir)
}

// Dont use eventloop and dont care about the tea.Cmd returned by Update()
func (p *TeaProg) SendDirectly(msgs ...tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(msgs))
	for i, msg := range msgs {
		var retModel tea.Model
		retModel, cmds[i] = p.m.Update(msg)
		if m, ok := retModel.(*model); ok {
			p.m = m
		} else {
			// This should never happen as we return *model on Update()
			panic("model is not of type *model")
		}
	}

	return tea.Batch(cmds...)
}

func (p *TeaProg) SendKeyDirectly(key string) tea.Cmd {
	return p.SendDirectly(utils.TeaRuneKeyMsg(key))
}

func (p *TeaProg) Close() {
	// Kill is asynchronous; Wait blocks until the event loop has fully stopped
	// so that later cleanups (e.g. restoring global config) never race with a
	// still-running event-loop goroutine.
	p.prog.Kill()
	p.prog.Wait()
}
