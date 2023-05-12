package readline

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	//tty "github.com/nyaosorg/go-readline-ny/tty10"
	"github.com/nyaosorg/go-readline-ny/internal/moji"
	"github.com/nyaosorg/go-readline-ny/keys"
	tty "github.com/nyaosorg/go-readline-ny/tty8"
)

// Result is the type for readline's result.
type Result int

const (
	// CONTINUE is returned by key-functions to continue the line editor
	CONTINUE Result = iota
	// ENTER is returned by key-functions when Enter key is pressed
	ENTER Result = iota
	// ABORT is returned by key-functions when Ctrl-D is pressed with no command-line
	ABORT Result = iota
	// INTR is returned by key-functions when Ctrl-C is pressed
	INTR Result = iota
)

// Editor is the main class to hold the parameter for ReadLine
type Editor struct {
	KeyMap
	History        IHistory
	Writer         io.Writer
	Out            *bufio.Writer
	Prompt         func() (int, error)
	PromptWriter   func(io.Writer) (int, error)
	Default        string
	Cursor         int
	LineFeed       func(Result)
	Tty            ITty
	Coloring       Coloring
	HistoryCycling bool
}

const (
	ansiCursorOff = "\x1B[?25l"

	// On Windows 8.1, the cursor is not shown immediately
	// without SetConsoleCursorPosition by `ESC[u`
	ansiCursorOn = "\x1B[?25h\x1B[s\x1B[u"
)

// CtrlC is the error when Ctrl-C is pressed.
var CtrlC = errors.New("^C")

var mu sync.Mutex

func (editor *Editor) LookupCommand(key string) Command {
	code := keys.Code(key)
	if editor.KeyMap.KeyMap != nil {
		if f, ok := editor.KeyMap.KeyMap[code]; ok {
			return f
		}
	}
	if f, ok := GlobalKeyMap.KeyMap[code]; ok {
		return f
	}
	return SelfInserter(key)
}

func (editor *Editor) printSimplePrompt() (int, error) {
	editor.Out.WriteString("\n> ")
	return 2, nil
}

func cutEscapeSequenceAndOldLine(s string) string {
	var buffer strings.Builder
	esc := false
	for i, end := 0, len(s); i < end; i++ {
		r := s[i]
		switch r {
		case '\r', '\n':
			buffer.Reset()
		case '\x1B':
			esc = true
		default:
			if esc {
				if ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z') {
					esc = false
				}
			} else {
				buffer.WriteByte(r)
			}
		}
	}
	return buffer.String()
}

func (editor *Editor) callPromptWriter() (int, error) {
	var buffer strings.Builder
	editor.PromptWriter(&buffer)
	prompt := buffer.String()
	_, err := editor.Out.WriteString(prompt)
	w, _ := moji.MojiWidthAndCountInString(cutEscapeSequenceAndOldLine(prompt))
	return int(w), err
}

// ReadLine calls LineEditor
// - ENTER typed -> returns TEXT and nil
// - CTRL-C typed -> returns "" and readline.CtrlC
// - CTRL-D typed -> returns "" and io.EOF
func (editor *Editor) ReadLine(ctx context.Context) (string, error) {
	if editor.Writer == nil {
		editor.Writer = os.Stdout
	}
	if editor.Out == nil {
		editor.Out = bufio.NewWriter(editor.Writer)
	}
	defer func() {
		editor.Out.WriteString(ansiCursorOn)
		editor.Out.Flush()
	}()

	if editor.Prompt == nil {
		if editor.PromptWriter != nil {
			editor.Prompt = editor.callPromptWriter
		} else {
			editor.Prompt = editor.printSimplePrompt
		}
	}
	if editor.History == nil {
		editor.History = _EmptyHistory{}
	}
	if editor.LineFeed == nil {
		editor.LineFeed = func(Result) {
			editor.Out.WriteByte('\n')
		}
	}
	if editor.Tty == nil {
		editor.Tty = &tty.Tty{}
	}
	buffer := Buffer{
		Editor:         editor,
		Buffer:         make([]Cell, 0, 20),
		historyPointer: editor.History.Len(),
	}

	if err := editor.Tty.Open(); err != nil {
		return "", fmt.Errorf("go-tty.Open: %s", err.Error())
	}
	defer editor.Tty.Close()

	var err error
	buffer.termWidth, _, err = editor.Tty.Size()
	if err != nil {
		return "", fmt.Errorf("go-tty.Size: %s", err.Error())
	}

	buffer.topColumn, err = editor.Prompt()
	if err != nil {
		// unable to get prompt-string.
		fmt.Fprintf(buffer.Out, "%s\n$ ", err.Error())
		buffer.topColumn = 2
	} else if buffer.topColumn >= buffer.termWidth-3 {
		// ViewWidth is too narrow to edit.
		io.WriteString(buffer.Out, "\n")
		buffer.topColumn = 0
	}
	buffer.InsertString(0, editor.Default)
	if buffer.Cursor > len(buffer.Buffer) {
		buffer.Cursor = len(buffer.Buffer)
	}
	buffer.RepaintAfterPrompt()
	buffer.startChangeWidthEventLoop(buffer.termWidth, editor.Tty.GetResizeNotifier())

	for {
		key, err := buffer.GetKey()
		if err != nil {
			return "", err
		}
		mu.Lock()

		f := editor.LookupCommand(key)

		io.WriteString(buffer.Out, ansiCursorOff)

		rc := f.Call(ctx, &buffer)

		io.WriteString(buffer.Out, ansiCursorOn)

		if rc != CONTINUE {
			buffer.LineFeed(rc)

			result := buffer.String()
			mu.Unlock()
			if rc == ENTER {
				return result, nil
			} else if rc == INTR {
				return result, CtrlC
			} else {
				return result, io.EOF
			}
		}
		mu.Unlock()
	}
}
