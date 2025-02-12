package ui

import (
	"strings"
	"sync/atomic"

	"git.sr.ht/~taiite/senpai/irc"

	"github.com/gdamore/tcell/v2"
)

type Config struct {
	NickColWidth   int
	ChanColWidth   int
	MemberColWidth int
	AutoComplete   func(cursorIdx int, text []rune) []Completion
	Mouse          bool
}

type UI struct {
	screen tcell.Screen
	Events chan tcell.Event
	exit   atomic.Value // bool
	config Config

	bs     BufferList
	e      Editor
	prompt StyledString
	status string

	memberOffset int
}

func New(config Config) (ui *UI, err error) {
	ui = &UI{
		config: config,
	}

	ui.screen, err = tcell.NewScreen()
	if err != nil {
		return
	}

	err = ui.screen.Init()
	if err != nil {
		return
	}
	if ui.screen.HasMouse() && config.Mouse {
		ui.screen.EnableMouse()
	}
	ui.screen.EnablePaste()

	_, h := ui.screen.Size()
	ui.screen.Clear()
	ui.screen.ShowCursor(0, h-2)

	ui.exit.Store(false)

	ui.Events = make(chan tcell.Event, 128)
	go func() {
		for !ui.ShouldExit() {
			ui.Events <- ui.screen.PollEvent()
		}
	}()

	ui.bs = NewBufferList()
	ui.e = NewEditor(ui.config.AutoComplete)
	ui.Resize()

	return
}

func (ui *UI) ShouldExit() bool {
	return ui.exit.Load().(bool)
}

func (ui *UI) Exit() {
	ui.exit.Store(true)
}

func (ui *UI) Close() {
	ui.screen.Fini()
}

func (ui *UI) CurrentBuffer() string {
	return ui.bs.Current()
}

func (ui *UI) NextBuffer() {
	ui.bs.Next()
	ui.memberOffset = 0
}

func (ui *UI) PreviousBuffer() {
	ui.bs.Previous()
	ui.memberOffset = 0
}

func (ui *UI) ClickedBuffer() int {
	return ui.bs.clicked
}

func (ui *UI) ClickBuffer(i int) {
	if i < len(ui.bs.list) {
		ui.bs.clicked = i
	}
}

func (ui *UI) GoToBufferNo(i int) {
	if ui.bs.To(i) {
		ui.memberOffset = 0
	}
}

func (ui *UI) ScrollUp() {
	ui.bs.ScrollUp(ui.bs.tlHeight / 2)
}

func (ui *UI) ScrollDown() {
	ui.bs.ScrollDown(ui.bs.tlHeight / 2)
}

func (ui *UI) ScrollUpBy(n int) {
	ui.bs.ScrollUp(n)
}

func (ui *UI) ScrollDownBy(n int) {
	ui.bs.ScrollDown(n)
}

func (ui *UI) ScrollMemberUpBy(n int) {
	ui.memberOffset -= n
	if ui.memberOffset < 0 {
		ui.memberOffset = 0
	}
}

func (ui *UI) ScrollMemberDownBy(n int) {
	ui.memberOffset += n
}

func (ui *UI) IsAtTop() bool {
	return ui.bs.IsAtTop()
}

func (ui *UI) AddBuffer(title string) (i int, added bool) {
	return ui.bs.Add(title)
}

func (ui *UI) RemoveBuffer(title string) {
	_ = ui.bs.Remove(title)
	ui.memberOffset = 0
}

func (ui *UI) AddLine(buffer string, notify NotifyType, line Line) {
	ui.bs.AddLine(buffer, notify, line)
}

func (ui *UI) AddLines(buffer string, before, after []Line) {
	ui.bs.AddLines(buffer, before, after)
}

func (ui *UI) JumpBuffer(sub string) bool {
	subLower := strings.ToLower(sub)
	for i, b := range ui.bs.list {
		if strings.Contains(strings.ToLower(b.title), subLower) {
			if ui.bs.To(i) {
				ui.memberOffset = 0
			}
			return true
		}
	}

	return false
}

func (ui *UI) JumpBufferIndex(i int) bool {
	if i >= 0 && i < len(ui.bs.list) {
		if ui.bs.To(i) {
			ui.memberOffset = 0
		}
		return true
	}
	return false
}

func (ui *UI) SetStatus(status string) {
	ui.status = status
}

func (ui *UI) SetPrompt(prompt StyledString) {
	ui.prompt = prompt
}

func (ui *UI) InputIsCommand() bool {
	return ui.e.IsCommand()
}

func (ui *UI) InputLen() int {
	return ui.e.TextLen()
}

func (ui *UI) InputRune(r rune) {
	ui.e.PutRune(r)
}

func (ui *UI) InputRight() {
	ui.e.Right()
}

func (ui *UI) InputRightWord() {
	ui.e.RightWord()
}

func (ui *UI) InputLeft() {
	ui.e.Left()
}

func (ui *UI) InputLeftWord() {
	ui.e.LeftWord()
}

func (ui *UI) InputHome() {
	ui.e.Home()
}

func (ui *UI) InputEnd() {
	ui.e.End()
}

func (ui *UI) InputUp() {
	ui.e.Up()
}

func (ui *UI) InputDown() {
	ui.e.Down()
}

func (ui *UI) InputBackspace() (ok bool) {
	return ui.e.RemRune()
}

func (ui *UI) InputDelete() (ok bool) {
	return ui.e.RemRuneForward()
}

func (ui *UI) InputDeleteWord() (ok bool) {
	return ui.e.RemWord()
}

func (ui *UI) InputAutoComplete(offset int) (ok bool) {
	return ui.e.AutoComplete(offset)
}

func (ui *UI) InputEnter() (content string) {
	return ui.e.Flush()
}

func (ui *UI) InputClear() bool {
	return ui.e.Clear()
}

func (ui *UI) InputBackSearch() {
	ui.e.BackSearch()
}

func (ui *UI) Resize() {
	w, h := ui.screen.Size()
	innerWidth := w - 9 - ui.config.ChanColWidth - ui.config.NickColWidth - ui.config.MemberColWidth
	ui.e.Resize(innerWidth)
	if ui.config.ChanColWidth == 0 {
		ui.bs.ResizeTimeline(innerWidth, h-3)
	} else {
		ui.bs.ResizeTimeline(innerWidth, h-2)
	}
}

func (ui *UI) Size() (int, int) {
	return ui.screen.Size()
}

func (ui *UI) Draw(members []irc.Member) {
	w, h := ui.screen.Size()

	if ui.config.ChanColWidth == 0 {
		ui.e.Draw(ui.screen, 9+ui.config.NickColWidth, h-2)
	} else {
		ui.e.Draw(ui.screen, 9+ui.config.ChanColWidth+ui.config.NickColWidth, h-1)
	}

	ui.bs.DrawTimeline(ui.screen, ui.config.ChanColWidth, 0, ui.config.NickColWidth)
	if ui.config.ChanColWidth == 0 {
		ui.bs.DrawHorizontalBufferList(ui.screen, 0, h-1, w-ui.config.MemberColWidth)
	} else {
		ui.bs.DrawVerticalBufferList(ui.screen, 0, 0, ui.config.ChanColWidth, h)
	}
	if ui.config.MemberColWidth != 0 {
		ui.bs.DrawVerticalMemberList(ui.screen, w-ui.config.MemberColWidth, 0, ui.config.MemberColWidth, h, members, &ui.memberOffset)
	}
	if ui.config.ChanColWidth == 0 {
		ui.drawStatusBar(ui.config.ChanColWidth, h-3, w-ui.config.MemberColWidth)
	} else {
		ui.drawStatusBar(ui.config.ChanColWidth, h-2, w-ui.config.ChanColWidth-ui.config.MemberColWidth)
	}

	if ui.config.ChanColWidth == 0 {
		for x := 0; x < 9+ui.config.NickColWidth; x++ {
			ui.screen.SetContent(x, h-2, ' ', nil, tcell.StyleDefault)
		}
		printIdent(ui.screen, 7, h-2, ui.config.NickColWidth, ui.prompt)
	} else {
		for x := ui.config.ChanColWidth; x < 9+ui.config.ChanColWidth+ui.config.NickColWidth; x++ {
			ui.screen.SetContent(x, h-1, ' ', nil, tcell.StyleDefault)
		}
		printIdent(ui.screen, ui.config.ChanColWidth+7, h-1, ui.config.NickColWidth, ui.prompt)
	}

	ui.screen.Show()
}

func (ui *UI) drawStatusBar(x0, y, width int) {
	width--

	st := tcell.StyleDefault.Dim(true)

	for x := x0; x < x0+width; x++ {
		ui.screen.SetContent(x, y, ' ', nil, st)
	}

	if ui.status == "" {
		return
	}

	var s StyledStringBuilder
	s.SetStyle(tcell.StyleDefault.Foreground(tcell.ColorGray))
	s.WriteString("--")

	x := x0 + 5 + ui.config.NickColWidth
	printString(ui.screen, &x, y, s.StyledString())
	x += 2

	s.Reset()
	s.WriteString(ui.status)

	printString(ui.screen, &x, y, s.StyledString())
}
