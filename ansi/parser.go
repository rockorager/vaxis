package ansi

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/rockorager/go-uucode"
)

const eof rune = -1

const (
	MaxIntermediate = 4
	MaxCSIParams    = 24
	InlineCSIParams = 12
)

var asciiPrint [128]Print

func init() {
	for r := rune(0x20); r < 0x7f; r++ {
		asciiPrint[r] = Print{Grapheme: string(r), Width: 1}
	}
}

// https://vt100.net/emu/dec_ansi_parser
//
// parser is an implementation of Paul Flo Williams' VT500-series
// parser, as seen [here](https://vt100.net/emu/dec_ansi_parser). The
// architecture is designed after Rob Pike's text/template parser, with a
// few modifications.
//
// Many of the comments are directly from Paul Flo Williams description of
// the parser, licensed undo [CC-BY-4.0](https://creativecommons.org/licenses/by/4.0/)
type Parser struct {
	close           chan bool
	closed          chan bool
	r               *bufio.Reader
	sequences       chan Sequence
	state           stateFn
	mode            ParserMode
	exit            func()
	intermediate    [MaxIntermediate]rune
	intermediateLen int
	params          [MaxCSIParams]uint32
	paramsLen       int
	paramsColon     uint32
	paramsOverflow  bool
	paramAcc        uint32
	paramDigits     int
	paramEmpty      bool
	final           rune

	// we turn ignoreST on when we enter a state that can "only" be exited
	// by ST. This will have the effect of ignore an ST so we don't see
	// ambiguous "Alt+\" when parsing input
	ignoreST bool

	// escTimeout is a timeout for interpretting an Esc keypress vs an
	// escape sequence
	escTimeout *time.Timer
	mu         sync.Mutex

	oscData        []rune
	oscInvalidUTF8 bool
	apcData        []rune

	dcs             DCS
	lastRuneInvalid bool
}

// ParserMode controls how ambiguous parser input is interpreted.
type ParserMode int

const (
	// ParserModeInput parses terminal input, where a bare ESC keypress must be
	// disambiguated from the start of a longer escape sequence.
	ParserModeInput ParserMode = iota
	// ParserModeOutput parses application output, where ESC always starts an
	// output control sequence and should not be emitted by a timer.
	ParserModeOutput
)

func NewParser(r io.Reader, modes ...ParserMode) *Parser {
	mode := ParserModeInput
	if len(modes) > 0 {
		mode = modes[0]
	}
	parser := &Parser{
		close:     make(chan bool, 1),
		closed:    make(chan bool, 1),
		r:         bufio.NewReader(r),
		sequences: make(chan Sequence, 2),
		state:     ground,
		mode:      mode,
	}
	// Rob Pike didn't use concurrency since he wanted templates to be able
	// to happen in init() functions, but we don't care about that.
	go parser.run()
	return parser
}

// Next returns the next Sequence. Sequences will be of the following types:
//
//	error Sent on any parsing error
//	Print Print the character to the screen
//	C0    Execute the C0 code
//	ESC   Execute the ESC sequence
//	CSI   Execute the CSI sequence
//	OSC   Execute the OSC sequence
//	DCS   Execute the DCS sequence
//	EOF   Sent at end of input
func (p *Parser) Next() <-chan Sequence {
	return p.sequences
}

// Finish is retained for compatibility with older callers. Sequences no longer
// hold pooled parser storage, so there is nothing to release.
func (p *Parser) Finish(Sequence) {
}

func (p *Parser) run() {
outer:
	for {
		select {
		case <-p.close:
			break outer
		default:
			r := p.readRune()
			p.mu.Lock()
			p.state = anywhere(r, p)
			if p.state == nil {
				p.mu.Unlock()
				break outer
			}
			p.mu.Unlock()
		}
	}
	if p.escTimeout != nil {
		p.escTimeout.Stop()
	}
	p.emit(EOF{})
	close(p.sequences)
	p.closed <- true
}

func (p *Parser) Close() {
	p.close <- true
}

func (p *Parser) WaitClose() {
	<-p.closed
}

func (p *Parser) readRune() rune {
	p.lastRuneInvalid = false
	r, _, err := p.r.ReadRune()
	if p.escTimeout != nil {
		p.escTimeout.Stop()
	}
	if r == unicode.ReplacementChar {
		// If invalid UTF-8, let's read the byte and deliver
		// it as is
		err = p.r.UnreadRune()
		if err != nil {
			return eof
		}
		b, err := p.r.ReadByte()
		if err != nil {
			return eof
		}
		r = rune(b)
		p.lastRuneInvalid = true
	}
	if err != nil {
		return eof
	}
	return r
}

func (p *Parser) emit(seq Sequence) {
	p.sequences <- seq
}

// This action only occurs in ground state. The current code should be mapped to
// a glyph according to the character set mappings and shift states in effect,
// and that glyph should be displayed. 20 (SP) and 7F (DEL) have special
// behaviour in later VT series, as described in ground.
func (p *Parser) print(r rune) {
	if r >= 0x20 && r < 0x7f {
		if p.r.Buffered() == 0 {
			p.emit(asciiPrint[r])
			return
		}
		var state uucode.BreakState
		next, _, err := p.r.ReadRune()
		if err != nil {
			p.emit(asciiPrint[r])
			return
		}
		if uucode.IsBreak(r, next, &state) {
			p.r.UnreadRune()
			p.emit(asciiPrint[r])
			return
		}

		bldr := strings.Builder{}
		bldr.WriteRune(r)
		bldr.WriteRune(next)
		prev := next
		for p.r.Buffered() > 0 {
			next, _, err := p.r.ReadRune()
			if err != nil {
				break
			}
			if uucode.IsBreak(prev, next, &state) {
				p.r.UnreadRune()
				break
			}
			bldr.WriteRune(next)
			prev = next
		}
		grapheme := bldr.String()
		p.emit(Print{Grapheme: grapheme, Width: uucode.StringWidth(grapheme)})
		return
	}

	bldr := strings.Builder{}
	bldr.WriteRune(r)

	prev := r
	var state uucode.BreakState
	for p.r.Buffered() > 0 {
		next, _, err := p.r.ReadRune()
		if err != nil {
			break
		}
		if uucode.IsBreak(prev, next, &state) {
			p.r.UnreadRune()
			break
		}
		bldr.WriteRune(next)
		prev = next
	}

	grapheme := bldr.String()
	p.emit(Print{Grapheme: grapheme, Width: uucode.StringWidth(grapheme)})
}

// The C0 or C1 control function should be executed, which may have any one of a
// variety of effects, including changing the cursor position, suspending or
// resuming communications or changing the shift states in effect. There are no
// parameters to this action.
func (p *Parser) execute(r rune) {
	if in(r, 0x00, 0x1F) {
		p.emit(C0(r))
		return
	}
}

// This action causes the current private flag, intermediate characters, final
// character and parameters to be forgotten. This occurs on entry to the escape,
// csi entry and dcs entry states, so that erroneous sequences like CSI 3 ; 1
// CSI 2 J are handled correctly.
func (p *Parser) clear() {
	p.final = rune(0)
	p.intermediateLen = 0
	p.paramsLen = 0
	p.paramsColon = 0
	p.paramsOverflow = false
	p.paramAcc = 0
	p.paramDigits = 0
	p.paramEmpty = false
}

// The private marker or intermediate character should be stored for later
// use in selecting a control function to be executed when a final
// character arrives. X3.64 doesn’t place any limit on the number of
// intermediate characters allowed before a final character, although it
// doesn’t define any control sequences with more than one. Digital defined
// escape sequences with two intermediate characters, and control sequences
// and device control strings with one. If more than two intermediate
// characters arrive, the parser can just flag this so that the dispatch
// can be turned into a null operation.
func (p *Parser) collect(r rune) {
	if p.intermediateLen >= MaxIntermediate {
		return
	}
	p.intermediate[p.intermediateLen] = r
	p.intermediateLen += 1
}

// The final character of an escape sequence has arrived, so determined the
// control function to be executed from the intermediate character(s) and
// final character, and execute it. The intermediate characters are
// available because collect stored them as they arrived.
func (p *Parser) escapeDispatch(r rune) {
	esc := ESC{
		Final: r,
	}
	esc.NumIntermediate = p.intermediateLen
	copy(esc.Intermediate[:], p.intermediate[:p.intermediateLen])
	p.emit(esc)
}

// This action collects the characters of a parameter string for a control
// sequence or device control sequence and builds a list of parameters. The
// characters processed by this action are the digits 0-9 (codes 30-39) and
// the semicolon (code 3B). The semicolon separates parameters. There is no
// limit to the number of characters in a parameter string, although a
// maximum of MaxCSIParams parameters need be stored. If more parameters
// arrive, all the extra parameters are silently ignored.
//
// Most control functions support default values for their parameters. The
// default value for a parameter is given by either leaving the parameter
// blank, or specifying a value of zero. Judging by previous threads on the
// newsgroup comp.terminals, this causes some confusion, with the
// occasional assertion that zero is the default parameter value for
// control functions. This is not the case: many control functions have a
// default value of 1, one (GSM) has a default value of 100, and some have
// no default. However, in all cases the default value is represented by
// either zero or a blank value.
//
// In the standard ECMA-48, which can be considered X3.64’s successor²,
// there is a distinction between a parameter with an empty value
// (representing the default value), and one that has the value zero. There
// used to be a mode, ZDM (Zero Default Mode), in which the two cases were
// treated identically, but that is now deprecated in the fifth edition
// (1991). Although a VT500 parser needs to treat both empty and zero
// parameters as representing the default, it is worth considering future
// extensions by distinguishing them internally
func (p *Parser) param(r rune) {
	switch r {
	case ';', ':':
		p.addParam()
		if r == ':' && p.paramsLen > 0 {
			p.paramsColon |= 1 << uint(p.paramsLen-1)
		}
		p.paramAcc = 0
		p.paramDigits = 0
		p.paramEmpty = true
	default:
		digit := uint32(r) - 0x30
		if p.paramAcc > (^uint32(0)-digit)/10 {
			p.paramAcc = ^uint32(0)
		} else {
			p.paramAcc *= 10
			p.paramAcc += digit
		}
		p.paramDigits += 1
		p.paramEmpty = false
	}
}

func (p *Parser) addParam() {
	if p.paramsLen >= MaxCSIParams {
		p.paramsOverflow = true
		return
	}
	p.params[p.paramsLen] = p.paramAcc
	p.paramsLen += 1
}

// A final character has arrived, so determine the control function to be
// executed from private marker, intermediate character(s) and final
// character, and execute it, passing in the parameter list.
func (p *Parser) csiDispatch(r rune) {
	csi := CSI{
		Final: r,
	}
	csi.NumIntermediate = p.intermediateLen
	copy(csi.Intermediate[:], p.intermediate[:p.intermediateLen])
	if p.paramDigits > 0 || p.paramEmpty || p.paramsLen > 0 {
		p.addParam()
	}
	if p.paramsOverflow {
		return
	}
	if p.mode == ParserModeOutput && p.paramsColon != 0 && r != 'm' {
		return
	}
	csi.NumParameters = p.paramsLen
	csi.ColonSeparators = p.paramsColon
	if p.paramsLen <= InlineCSIParams {
		copy(csi.Parameters[:], p.params[:p.paramsLen])
	} else {
		csi.ExtraParameters = make([]uint32, p.paramsLen)
		copy(csi.ExtraParameters, p.params[:p.paramsLen])
	}
	p.emit(csi)
}

// When the control function OSC (Operating System Command) is recognised,
// this action initializes an external parser (the “OSC Handler”) to handle
// the characters from the control string. OSC control strings are not
// structured in the same way as device control strings, so there is no
// choice of parsers.
//
// oscStart registers oscEnd as the exit function. This will be called on when
// the state moves from oscString to any other state
func (p *Parser) oscStart() {
	// p.emit(OSCStart{})
	p.oscInvalidUTF8 = false
	p.ignoreST = true
	p.exit = p.oscEnd
}

// This action passes characters from the control string to the OSC Handler
// as they arrive. There is therefore no need to buffer characters until
// the end of the control string is recognised.
func (p *Parser) oscPut(r rune) {
	if p.lastRuneInvalid {
		p.oscInvalidUTF8 = true
	}
	p.oscData = append(p.oscData, r)
	// p.emit(OSCData(r))
}

// This action is called when the OSC string is terminated by ST, CAN, SUB
// or ESC, to allow the OSC handler to finish neatly.
func (p *Parser) oscEnd() {
	p.emit(OSC{
		Payload:     p.oscData,
		InvalidUTF8: p.oscInvalidUTF8,
	})
	// OSC will usually be a hyperlink or pasted text, these can be pretty
	// large so we'll initialize with 128
	p.oscData = make([]rune, 0, 128)
	p.oscInvalidUTF8 = false
}

// This action is invoked when a final character arrives in the first part
// of a device control string. It determines the control function from the
// private marker, intermediate character(s) and final character, and
// executes it, passing in the parameter list. It also selects a handler
// function for the rest of the characters in the control string. This
// handler function will be called by the put action for every character in
// the control string as it arrives.
//
// This way of handling device control strings has been selected because it
// allows the simple plugging-in of extra parsers as functionality is
// added. Support for a fairly simple control string like DECDLD (Downline
// Load) could be added into the main parser if soft characters were
// required, but the main parser is no place for complicated protocols like
// ReGIS.
//
// hook registers unhook as the exit function. This will be called on when
// the state moves from dcsPassthrough to any other state.
func (p *Parser) hook(r rune) stateFn {
	if p.paramDigits > 0 || p.paramEmpty || p.paramsLen > 0 {
		p.addParam()
	}
	if p.paramsOverflow {
		p.dcs = DCS{}
		return dcsIgnore
	}

	p.ignoreST = true
	p.exit = p.unhook
	p.dcs = DCS{
		Final: r,
		Data:  make([]rune, 0, 128),
	}
	p.dcs.NumIntermediate = p.intermediateLen
	copy(p.dcs.Intermediate[:], p.intermediate[:p.intermediateLen])
	p.dcs.NumParameters = p.paramsLen
	if p.paramsLen <= InlineCSIParams {
		copy(p.dcs.Parameters[:], p.params[:p.paramsLen])
	} else {
		p.dcs.ExtraParameters = make([]uint32, p.paramsLen)
		copy(p.dcs.ExtraParameters, p.params[:p.paramsLen])
	}
	return dcsPassthrough
}

// This action passes characters from the data string part of a device
// control string to a handler that has previously been selected by the
// hook action. C0 controls are also passed to the handler.
func (p *Parser) put(r rune) {
	p.dcs.Data = append(p.dcs.Data, r)
}

// When a device control string is terminated by ST, CAN, SUB or ESC, this
// action calls the previously selected handler function with an “end of
// data” parameter. This allows the handler to finish neatly.
func (p *Parser) unhook() {
	p.emit(p.dcs)
	p.dcs = DCS{}
}

func (p *Parser) apcUnhook() {
	p.emit(APC{
		Data: string(p.apcData),
	})
	p.apcData = []rune{}
}

// in returns true if the rune lies within the range, inclusive of the endpoints
func in(r rune, min int32, max int32) bool {
	if r >= min && r <= max {
		return true
	}
	return false
}

// State functions

type stateFn func(rune, *Parser) stateFn

// This isn’t a real state. It is used on the state diagram to show
// transitions that can occur from any state to some other state.
func anywhere(r rune, p *Parser) stateFn {
	switch {
	case r == eof:
		if p.exit != nil {
			p.exit()
			p.exit = nil
		}
		return nil
	case r == 0x18, r == 0x1A:
		if p.exit != nil {
			p.exit()
			p.exit = nil
		}
		p.execute(r)
		return ground
	case in(r, 0x80, 0x9F):
		return c1Control(r, p)
	case r == 0x1B:
		if p.exit != nil {
			p.exit()
			p.exit = nil
		}
		p.clear()
		if p.mode == ParserModeInput {
			p.escTimeout = time.AfterFunc(10*time.Millisecond, func() {
				p.emit(C0(0x1B))
				p.mu.Lock()
				p.state = ground
				p.mu.Unlock()
			})
		}
		return escape
	default:
		return p.state(r, p)
	}
}

func c1Control(r rune, p *Parser) stateFn {
	if p.exit != nil {
		p.exit()
		p.exit = nil
	}
	p.ignoreST = false
	p.clear()

	switch r {
	case 0x90: // DCS
		return dcsEntry
	case 0x98, 0x9E: // SOS, PM
		p.ignoreST = true
		return sosPm
	case 0x9B: // CSI
		return csiEntry
	case 0x9C: // ST
		return ground
	case 0x9D: // OSC
		p.oscStart()
		return oscString
	case 0x9F: // APC
		p.ignoreST = true
		p.exit = p.apcUnhook
		return apc
	default:
		p.escapeDispatch(r - 0x40)
		return ground
	}
}

// This state is entered when the control function CSI is recognised, in
// 7-bit or 8-bit form. This state will only deal with the first character
// of a control sequence, because the characters 3C-3F can only appear as
// the first character of a control sequence, if they appear at all.
// Strictly speaking, X3.64 says that the entire string is “subject to
// private or experimental interpretation” if the first character is one of
// 3C-3F, which allows sequences like CSI ?::<? F, but Digital’s terminals
// only ever used one private-marker character at a time. As far as I am
// aware, only characters 3D (=), 3E (>) and 3F (?) were used by Digital.
//
// C0 controls are executed immediately during the recognition of a control
// sequence. C1 controls will cancel the sequence and then be executed. I
// imagine this treatment of C1 controls is prompted by the consideration
// that the 7-bit (ESC Fe) and 8-bit representations of C1 controls should
// act in the same way. When the first character of the 7-bit
// representation, ESC, is received, it will cancel the control sequence,
// so the 8-bit representation should do so as well.
func csiEntry(r rune, p *Parser) stateFn {
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		p.execute(r)
		return csiEntry
	case r == 0x7F:
		// ignore
		return csiEntry
	case in(r, 0x30, 0x39), r == 0x3B, r == 0x3A:
		// 0x3A is not per the PFW, but using colons is valid SGR
		// syntax for separating params when including colorspace. The
		// colorspace should be ignored
		p.param(r)
		return csiParam
	case in(r, 0x3C, 0x3F):
		p.collect(r)
		return csiParam
	// case is(r, 0x3A):
	// 	return csiIgnore
	case in(r, 0x20, 0x2F):
		p.collect(r)
		return csiIntermediate
	case in(r, 0x40, 0x7E):
		p.csiDispatch(r)
		return ground
	default:
		// Return to ground on unexpected characters
		p.emit(fmt.Errorf("unexpected characted: %c", r))
		return ground
	}
}

// This state is entered when a parameter character is recognised in a
// control sequence. It then recognises other parameter characters until an
// intermediate or final character appears. Further occurrences of the
// private-marker characters 3C-3F or the character 3A, which has no
// standardised meaning, will cause transition to the csi ignore state.
func csiParam(r rune, p *Parser) stateFn {
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		p.execute(r)
		return csiParam
	case r == 0x7F:
		// ignore
		return csiParam
	case in(r, 0x30, 0x39), r == 0x3B, r == 0x3A:
		// 0x3A is not per the PFW, but using colons is valid SGR
		// syntax for separating params when including colorspace. The
		// colorspace should be ignored
		p.param(r)
		return csiParam
	case in(r, 0x40, 0x7E):
		p.csiDispatch(r)
		return ground
	case in(r, 0x20, 0x2F):
		p.collect(r)
		return csiIntermediate
	case in(r, 0x3C, 0x3F):
		return csiIgnore
	default:
		// Return to ground on unexpected characters
		p.emit(fmt.Errorf("unexpected characted: %c", r))
		return ground
	}
}

// This state is used to consume remaining characters of a control sequence
// that is still being recognised, but has already been disregarded as
// malformed. This state will only exit when a final character is
// recognised, at which point it transitions to ground state without
// dispatching the control function. This state may be entered because:
//
//  1. a private-marker character 3C-3F is recognised in any place other
//     than the first character of the control sequence,
//  2. the character 3A appears anywhere, or
//  3. a parameter character 30-3F occurs after an intermediate
//     character has been recognised.
//
// C0 controls will still be executed while a control sequence is being
// ignored
func csiIgnore(r rune, p *Parser) stateFn {
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		p.execute(r)
		return csiIgnore
	case r == 0x7F:
		// ignore
		return csiIgnore
	case in(r, 0x40, 0x7E):
		return ground
	default:
		return csiIgnore
	}
}

// This state is entered when an intermediate character is recognised in a
// control sequence. It then recognises other intermediate characters until
// a final character appears. If any more parameter characters appear, this
// is an error condition which will cause a transition to the csi ignore
// state.
func csiIntermediate(r rune, p *Parser) stateFn {
	switch {
	case r == eof:
		return nil
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		p.execute(r)
		return csiIntermediate
	case r == 0x7F:
		// ignore
		return csiIntermediate
	case in(r, 0x20, 0x2F):
		p.collect(r)
		return csiIntermediate
	case in(r, 0x30, 0x3F):
		return csiIgnore
	case in(r, 0x40, 0x7E):
		p.csiDispatch(r)
		return ground
	default:
		// Return to ground on unexpected characters
		p.emit(fmt.Errorf("unexpected characted: %c", r))
		return ground
	}
}

// This state is entered when the control function DCS is recognised, in
// 7-bit or 8-bit form. X3.64 doesn’t define any structure for device
// control strings, but Digital made them appear like control sequences
// followed by a data string, with a form and length dependent on the
// control function. This state is only used to recognise the first
// character of the control string, mirroring the csi entry state.
//
// C0 controls other than CAN, SUB and ESC are not executed while
// recognising the first part of a device control string.
func dcsEntry(r rune, p *Parser) stateFn {
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		// ignore
		return dcsEntry
	case r == 0x7F:
		// ignore
		return dcsEntry
	case in(r, 0x20, 0x2F):
		p.collect(r)
		return dcsIntermediate
	case r == 0x3A:
		return dcsIgnore
	case in(r, 0x30, 0x39), r == 0x3B:
		p.param(r)
		return dcsParam
	case in(r, 0x3C, 0x3F):
		p.collect(r)
		return dcsParam
	case in(r, 0x40, 0x7E):
		return p.hook(r)
	default:
		return p.hook(r)
	}
}

// This state is entered when an intermediate character is recognised in a
// device control string. It then recognises other intermediate characters
// until a final character appears. If any more parameter characters
// appear, this is an error condition which will cause a transition to the
// dcs ignore state.
func dcsIntermediate(r rune, p *Parser) stateFn {
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		// ignore
		return dcsIntermediate
	case in(r, 0x20, 0x2F):
		p.collect(r)
		return dcsIntermediate
	case r == 0x7F:
		// ignore
		return dcsIntermediate
	case in(r, 0x30, 0x3F):
		return dcsIgnore
	case in(r, 0x40, 0x7E):
		return p.hook(r)
	default:
		// Return to ground on unexpected characters
		p.emit(fmt.Errorf("unexpected characted: %c", r))
		return ground
	}
}

// This state is entered when a parameter character is recognised in a
// device control string. It then recognises other parameter characters
// until an intermediate or final character appears. Occurrences of the
// private-marker characters 3C-3F or the undefined character 3A will cause
// a transition to the dcs ignore state.
func dcsParam(r rune, p *Parser) stateFn {
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		// ignore
		return dcsParam
	case in(r, 0x30, 0x39), r == 0x3B:
		p.param(r)
		return dcsParam
	case r == 0x7F:
		// ignore
		return dcsParam
	case in(r, 0x20, 0x2F):
		p.collect(r)
		return dcsIntermediate
	case r == 0x3A, in(r, 0x3C, 0x3F):
		return dcsIgnore
	case in(r, 0x40, 0x7E):
		return p.hook(r)
	default:
		// Return to ground on unexpected characters
		p.emit(fmt.Errorf("unexpected characted: %c", r))
		return ground
	}
}

// This state is used to consume remaining characters of a device control
// string that is still being recognised, but has already been disregarded
// as malformed. This state will only exit when the control function ST is
// recognised, at which point it transitions to ground state. This state
// may be entered because:
//
//  1. a private-marker character 3C-3F is recognised in any place other
//     than the first character of the control string,
//  2. the character 3A appears anywhere, or
//  3. a parameter character 30-3F occurs after an intermediate
//     character has been recognised.
//
// These conditions are only errors in the first part of the control
// string, until a final character has been recognised. The data string
// that follows is not checked by this parser.
func dcsIgnore(r rune, p *Parser) stateFn {
	p.ignoreST = true
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		// ignore
		return dcsIgnore
	case in(r, 0x20, 0x7F):
		// ignore
		return dcsIgnore
	default:
		// ignore
		return dcsIgnore
	}
}

// This state is a shortcut for writing state machines for all possible
// device control strings into the main parser. When a final character has
// been recognised in a device control string, this state will establish a
// channel to a handler for the appropriate control function, and then pass
// all subsequent characters through to this alternate handler, until the
// data string is terminated (usually by recognising the ST control
// function).
//
// This state has an exit action so that the control function handler can
// be informed when the data string has come to an end. This is so that the
// last soft character in a DECDLD string can be completed when there is no
// other means of knowing that its definition has ended, for example.
func dcsPassthrough(r rune, p *Parser) stateFn {
	p.ignoreST = true
	p.exit = p.unhook
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		p.put(r)
		return dcsPassthrough
	case in(r, 0x20, 0x7E):
		p.put(r)
		return dcsPassthrough
	case r == 0x7F:
		// ignore
		return dcsPassthrough
	default:
		p.put(r)
		return dcsPassthrough
	}
}

// This state is entered whenever the C0 control ESC is received. This will
// immediately cancel any escape sequence, control sequence or control
// string in progress. If an escape sequence or control sequence was in
// progress, “cancel” means that the sequence will have no effect, because
// the final character that determines the control function (in conjunction
// with any intermediates) will not have been received. However, the ESC
// that cancels a control string may occur after the control function has
// been determined and the following string has had some effect on terminal
// state. For example, some soft characters may already have been defined.
// Cancelling a control string does not undo these effects.
//
// A control string that started with DCS, OSC, PM or APC is usually
// terminated by the C1 control ST (String Terminator). In a 7-bit
// environment, ST will be represented by ESC \ (1B 5C). However, receiving
// the ESC character will “cancel” the control string, so the ST control
// function that is invoked by the arrival of the following “\” is
// essentially a “no-op” function. Does this point seem like pure trivia?
// Maybe, but I worried for ages about whether the control string
// recogniser needed a one character lookahead in order to know whether ESC
// \ was going to terminate it. The actual solution became clear when I was
// using ReGIS on a VT330: sending ESC immediately caused the graphics
// output cursor to disappear from the screen, so I knew that the control
// string had already finished before the “\” arrived. Many of the clues
// that enabled me to derive this state diagram have been as subtle as
// that.
func escape(r rune, p *Parser) stateFn {
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		p.execute(r)
		p.ignoreST = false
		return escape
	case in(r, 0x20, 0x2F):
		p.collect(r)
		p.ignoreST = false
		return escapeIntermediate
	case in(r, 0x30, 0x4E),
		in(r, 0x51, 0x57),
		r == 0x59,
		r == 0x5A,
		in(r, 0x60, 0x7F): // 0x7F is included here to allow for Alt+BackSpace inputs
		p.escapeDispatch(r)
		p.ignoreST = false
		return ground
	case r == 0x5C:
		if p.ignoreST {
			p.ignoreST = false
			return ground
		}
		p.escapeDispatch(r)
		p.ignoreST = false
		return ground
	case r == 0x4F:
		p.ignoreST = false
		return ss3
	case r == 0x50:
		p.clear()
		p.ignoreST = false
		return dcsEntry
	case r == 0x58, r == 0x5E:
		p.ignoreST = true
		return sosPm
	case r == 0x5F:
		p.ignoreST = true
		p.exit = p.apcUnhook
		return apc
	case r == 0x5B:
		p.clear()
		p.ignoreST = false
		return csiEntry
	case r == 0x5D:
		p.oscStart()
		return oscString
	default:
		// Return to ground on unexpected characters
		p.ignoreST = false
		return ground
	}
}

func ss3(r rune, p *Parser) stateFn {
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		p.execute(r)
		return ss3
	case r == 0x7F:
		// ignore
		return ss3
	default:
		p.emit(SS3(r))
		return ground
	}
}

// This state is entered when an intermediate character arrives in an
// escape sequence. Escape sequences have no parameters, so the control
// function to be invoked is determined by the intermediate and final
// characters. In this parser there is just one escape intermediate, and
// the parser uses the collect action to remember intermediate characters
// as they arrive, for processing by the esc_dispatch action when the final
// character arrives. An alternate approach (and the one adopted by xterm)
// is to have multiple copies of this state and choose the next appropriate
// one as each intermediate character arrives. I think that this alternate
// approach is merely an optimisation; the approach presented here doesn’t
// require any more states if the repertoire of supported control functions
// increases.
//
// This state is only split from the escape state because certain escape
// sequences are the 7-bit representations of C1 controls that change the
// state of the parser. Without these “compatibility sequences”, there
// could just be one escape state to collect intermediates and dispatch the
// sequence when a final character was received.
func escapeIntermediate(r rune, p *Parser) stateFn {
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		p.execute(r)
		return escapeIntermediate
	case r == 0x7F:
		// ignore
		return escapeIntermediate
	case in(r, 0x20, 0x2F):
		p.collect(r)
		return escapeIntermediate
	case in(r, 0x30, 0x7E):
		p.escapeDispatch(r)
		return ground
	default:
		// Return to ground on unexpected characters
		return ground
	}
}

// The VT500 doesn’t define any function for these control strings, so this
// state ignores all received characters until the control function ST is
// recognised.
func sosPm(r rune, p *Parser) stateFn {
	p.ignoreST = true
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		// ignore
		return sosPm
	default:
		return sosPm
	}
}

func apc(r rune, p *Parser) stateFn {
	p.ignoreST = true
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		// ignore
		return apc
	default:
		p.apcData = append(p.apcData, r)
		return apc
	}
}

// This is the initial state of the parser, and the state used to consume
// all characters other than components of escape and control sequences.
//
// GL characters (20 to 7F) are printed. I have included 20 (SP) and 7F
// (DEL) in this area, although both codes have special behaviour. If a
// 94-character set is mapped into GL, 20 will cause a space to be
// displayed, and 7F will be ignored. When a 96-character set is mapped
// into GL, both 20 and 7F may cause a character to be displayed. Later
// models of the VT220 included the DEC Multinational Character Set (MCS),
// which has 94 characters in its supplemental set (i.e. the characters
// supplied in addition to ASCII), so terminals only claiming VT220
// compatibility can always ignore 7F. The VT320 introduced ISO Latin-1,
// which has 96 characters in its supplemental set, so emulators with a
// VT320 compatibility mode need to treat 7F as a printable character.
func ground(r rune, p *Parser) stateFn {
	switch {
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		p.execute(r)
		return ground
	default:
		p.print(r)
		return ground
	}
}

// This state is entered when the control function OSC (Operating System
// Command) is recognised. On entry it prepares an external parser for OSC
// strings and passes all printable characters to a handler function. C0
// controls other than CAN, SUB and ESC are ignored during reception of the
// control string.
//
// The only control functions invoked by OSC strings are DECSIN (Set Icon
// Name) and DECSWT (Set Window Title), present on the multisession VT520
// and VT525 terminals. Earlier terminals treat OSC in the same way as PM
// and APC, ignoring the entire control string.
func oscString(r rune, p *Parser) stateFn {
	p.ignoreST = true
	switch {
	case r == 0x07:
		p.exit()
		p.exit = nil
		return ground
	case in(r, 0x00, 0x17), r == 0x19, in(r, 0x1C, 0x1F):
		// ignore
		return oscString
	case in(r, 0x20, 0x7F):
		p.oscPut(r)
		return oscString
	default:
		// catch all for UTF-8
		p.oscPut(r)
		return oscString
	}
}

// Sequence is the generic data type of items emitted from the parser. These can
// be control sequences, escape sequences, or printable characters.
type Sequence interface{}

// A character which should be printed to the screen
type Print struct {
	Grapheme string
	Width    int
}

func (seq Print) String() string {
	return fmt.Sprintf("Print: %q", seq.Grapheme)
}

// A C0 control code
type C0 rune

func (seq C0) String() string {
	return fmt.Sprintf("C0 0x%X", rune(seq))
}

// An escape sequence with intermediate characters
type ESC struct {
	Intermediate    [MaxIntermediate]rune
	NumIntermediate int
	Final           rune
}

func (seq ESC) Intermediates() []rune {
	return seq.Intermediate[:seq.NumIntermediate]
}

func (seq ESC) String() string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("ESC")
	for _, p := range seq.Intermediates() {
		buf.WriteRune(' ')
		buf.WriteRune(p)
	}
	buf.WriteRune(' ')
	buf.WriteRune(seq.Final)
	return buf.String()
}

type SS3 rune

func (seq SS3) String() string {
	return fmt.Sprintf("SS3 0x%X", rune(seq))
}

// A CSI Sequence
type CSI struct {
	Intermediate    [MaxIntermediate]rune
	NumIntermediate int
	Parameters      [InlineCSIParams]uint32
	ExtraParameters []uint32
	NumParameters   int
	ColonSeparators uint32
	Final           rune
}

func (seq CSI) Intermediates() []rune {
	return seq.Intermediate[:seq.NumIntermediate]
}

func (seq CSI) Params() []uint32 {
	if seq.NumParameters <= InlineCSIParams {
		return seq.Parameters[:seq.NumParameters]
	}
	return seq.ExtraParameters
}

func (seq CSI) Param(i int) int {
	if i < 0 || i >= seq.NumParameters {
		return 0
	}
	return int(seq.Params()[i])
}

func (seq CSI) ColonAfter(i int) bool {
	if i < 0 || i >= MaxCSIParams {
		return false
	}
	return seq.ColonSeparators&(1<<uint(i)) != 0
}

func (seq CSI) ParamGroup(i int) []uint32 {
	if i < 0 || i >= seq.NumParameters {
		return nil
	}
	end := i + 1
	for end < seq.NumParameters && seq.ColonAfter(end-1) {
		end += 1
	}
	return seq.Params()[i:end]
}

func (seq CSI) ParamGroupAt(i int) ([]uint32, int) {
	group := seq.ParamGroup(i)
	return group, i + len(group)
}

func (seq CSI) ParameterGroups() [][]int {
	if seq.NumParameters == 0 {
		return nil
	}
	groups := make([][]int, 0, seq.NumParameters)
	for i := 0; i < seq.NumParameters; {
		group, next := seq.ParamGroupAt(i)
		ints := make([]int, len(group))
		for j, v := range group {
			ints[j] = int(v)
		}
		groups = append(groups, ints)
		i = next
	}
	return groups
}

func (seq CSI) Command() string {
	intermediates := seq.Intermediates()
	buf := make([]rune, 0, len(intermediates)+1)
	buf = append(buf, intermediates...)
	buf = append(buf, seq.Final)
	return string(buf)
}

func (seq CSI) String() string {
	segments := make([]string, 0, 9)
	segments = append(segments, "CSI")
	intermediates := seq.Intermediates()
	if len(intermediates) > 0 {
		segments = append(segments, string(intermediates[0]))
	}
	params := seq.Params()
	for i, p := range params {
		if i > 0 {
			if seq.ColonAfter(i - 1) {
				segments = append(segments, ":")
			} else {
				segments = append(segments, ";")
			}
		}
		segments = append(segments, fmt.Sprintf("%d", p))
	}
	if len(intermediates) > 1 {
		segments = append(segments, string(intermediates[1:]))
	}
	segments = append(segments, string(seq.Final))
	return strings.Join(segments, " ")
}

// An OSC sequence. The Payload is the raw runes received, and must be parsed
// externally
type OSC struct {
	Payload     []rune
	InvalidUTF8 bool
}

func (seq OSC) String() string {
	return "OSC " + string(seq.Payload)
}

// Sent at the beginning of a DCS passthrough sequence.
type DCS struct {
	Final           rune
	Intermediate    [MaxIntermediate]rune
	NumIntermediate int
	Parameters      [InlineCSIParams]uint32
	ExtraParameters []uint32
	NumParameters   int
	Data            []rune
}

func (seq DCS) Intermediates() []rune {
	return seq.Intermediate[:seq.NumIntermediate]
}

func (seq DCS) Params() []uint32 {
	if seq.NumParameters <= InlineCSIParams {
		return seq.Parameters[:seq.NumParameters]
	}
	return seq.ExtraParameters
}

func (seq DCS) String() string {
	segments := make([]string, 0, 9)
	segments = append(segments, "DCS")
	intermediates := seq.Intermediates()
	if len(intermediates) > 0 {
		segments = append(segments, string(intermediates[0]))
	}
	for i, p := range seq.Params() {
		if i > 0 {
			segments = append(segments, ";")
		}
		segments = append(segments, fmt.Sprintf("%d", p))
	}
	if len(intermediates) > 1 {
		segments = append(segments, string(intermediates[1:]))
	}
	segments = append(segments, string(seq.Final))

	if len(seq.Data) > 0 {
		segments = append(segments, string(seq.Data))
	}
	return strings.Join(segments, " ")
}

type APC struct {
	Data string
}

func (seq APC) String() string {
	return "APC " + seq.Data
}

// Sent when the underlying PTY is closed
type EOF struct{}

func (seq EOF) String() string {
	return "EOF"
}
