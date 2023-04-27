package readline

import (
	"fmt"
	"unicode/utf8"

	"github.com/nyaosorg/go-readline-ny/internal/moji"
)

func (B *Buffer) refreshColor() ColorSequence {
	if B.Coloring == nil {
		B.Coloring = _MonoChrome{}
	}
	defaultColor := B.Coloring.Init()
	position := int16(0)
	for i, cell := range B.Buffer {
		B.Buffer[i].position = position
		if codepoint, ok := moji.MojiToRune(cell.Moji); ok {
			B.Buffer[i].color = ColorSequence(B.Coloring.Next(codepoint))
		} else {
			B.Buffer[i].color = ColorSequence(B.Coloring.Next(utf8.RuneError))
		}
		position += int16(cell.Moji.Width())
	}
	return defaultColor
}

// InsertAndRepaint inserts str and repaint the editline.
func (B *Buffer) InsertAndRepaint(str string) {
	B.ReplaceAndRepaint(B.Cursor, str)
}

// GotoHead move screen-cursor to the top of the viewarea.
// It should be called before text is changed.
func (B *Buffer) GotoHead() {
	fmt.Fprintf(B, "\x1B[%dG", B.topColumn+1)
}

func (B *Buffer) repaint() {
	all, left := B.view2()
	B.GotoHead()
	B.puts(all)
	B.eraseline()
	B.GotoHead()
	B.puts(left)
}

// DrawFromHead draw all text in viewarea and
// move screen-cursor to the position where it should be.
func (B *Buffer) DrawFromHead() {
	B.repaint()
}

// ReplaceAndRepaint replaces the string between `pos` and cursor's position to `str`
func (B *Buffer) ReplaceAndRepaint(pos int, str string) {
	// Replace Buffer
	B.Delete(pos, B.Cursor-pos)

	// Define ViewStart , Cursor
	B.Cursor = pos + B.InsertString(pos, str)
	B.joinUndo() // merge delete and insert
	B.ResetViewStart()
	B.repaint()
}

// RepaintAfterPrompt repaints the all characters in the editline except for prompt.
func (B *Buffer) RepaintAfterPrompt() {
	B.ResetViewStart()
	B.repaint()
}

// RepaintAll repaints the all characters in the editline including prompt.
func (B *Buffer) RepaintAll() {
	B.Out.Flush()
	B.topColumn, _ = B.Prompt()
	B.repaint()
}
