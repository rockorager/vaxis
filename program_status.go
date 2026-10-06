package vaxis

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ProgramState is the state reported by the Program Status Protocol (OSC
// 7501). See https://www.superlogical.com/rex/docs/build/program-status
type ProgramState string

const (
	// ProgramIdle means the program is at rest, waiting for the user's next
	// instruction.
	ProgramIdle ProgramState = "idle"
	// ProgramWorking means the program is running.
	ProgramWorking ProgramState = "working"
	// ProgramDone means the program finished a piece of work and the result
	// is ready to look at.
	ProgramDone ProgramState = "done"
	// ProgramBlocked means the program cannot continue until the user does
	// something.
	ProgramBlocked ProgramState = "blocked"
	// ProgramError means the program failed and stopped.
	ProgramError ProgramState = "error"
)

// ProgramBlockedKind says what a [ProgramBlocked] program is waiting for.
type ProgramBlockedKind string

const (
	// ProgramBlockedPermission means the program needs approval to do
	// something.
	ProgramBlockedPermission ProgramBlockedKind = "permission"
	// ProgramBlockedQuestion means the user must type an answer.
	ProgramBlockedQuestion ProgramBlockedKind = "question"
	// ProgramBlockedAuth means the program needs a login, token, or
	// credential.
	ProgramBlockedAuth ProgramBlockedKind = "auth"
)

const (
	// programStatusQuery is the feature detection body. A terminal replies
	// with the same body and may append pairs after the "?".
	programStatusQuery    = "7501;?"
	programStatusMaxTitle = 192
	programStatusMaxMsg   = 2048
)

// ProgramStatus is one Program Status Protocol report. Each report replaces
// its record completely, so include App and Title in every report that
// should carry them.
type ProgramStatus struct {
	// State is required.
	State ProgramState
	// ID addresses a record. The empty string addresses the root record.
	// Segments are separated by "/" and each matches [A-Za-z0-9_.+-]{1,32},
	// with at most 8 segments and 128 bytes in total.
	ID string
	// Kind is only sent with [ProgramBlocked].
	Kind ProgramBlockedKind
	// Progress is a percentage from 0 to 100. It is only sent with
	// [ProgramWorking] or [ProgramBlocked], and only when HasProgress is
	// true.
	Progress    int
	HasProgress bool
	// App is a stable, machine-readable program name matching
	// [A-Za-z0-9_.+-]{1,32}, for example "cargo".
	App string
	// Title is a short human-readable label for the record. Control
	// characters are removed and it is truncated to 192 bytes.
	Title string
	// Msg is one human-readable line describing the record. Control
	// characters are removed and it is truncated to 2048 bytes.
	Msg string
}

// SetProgramStatus reports what the program is doing using the Program Status
// Protocol (OSC 7501). Terminals which do not support the protocol ignore the
// sequence; use [Vaxis.CanProgramStatus] to check for support.
func (vx *Vaxis) SetProgramStatus(status ProgramStatus) error {
	body, err := status.encode()
	if err != nil {
		return err
	}
	vx.writeControlString(tparm(programStatus, body))
	return nil
}

// ClearProgramStatus removes the record addressed by id and every record
// beneath it. An empty id removes every record on the terminal.
func (vx *Vaxis) ClearProgramStatus(id string) error {
	body := "state=clear"
	if id != "" {
		if !validProgramStatusID(id) {
			return fmt.Errorf("vaxis: invalid program status id %q", id)
		}
		body += ":id=" + id
	}
	vx.writeControlString(tparm(programStatus, body))
	return nil
}

// CanProgramStatus reports whether the terminal replied to the Program Status
// Protocol (OSC 7501) feature detection query.
func (vx *Vaxis) CanProgramStatus() bool {
	vx.mu.Lock()
	defer vx.mu.Unlock()
	return vx.caps.programStatus
}

func (s ProgramStatus) encode() (string, error) {
	switch s.State {
	case ProgramIdle, ProgramWorking, ProgramDone, ProgramBlocked, ProgramError:
	default:
		return "", fmt.Errorf("vaxis: invalid program state %q", s.State)
	}
	pairs := []string{"state=" + string(s.State)}
	if s.ID != "" {
		if !validProgramStatusID(s.ID) {
			return "", fmt.Errorf("vaxis: invalid program status id %q", s.ID)
		}
		pairs = append(pairs, "id="+s.ID)
	}
	if s.State == ProgramBlocked && s.Kind != "" {
		switch s.Kind {
		case ProgramBlockedPermission, ProgramBlockedQuestion, ProgramBlockedAuth:
		default:
			return "", fmt.Errorf("vaxis: invalid program blocked kind %q", s.Kind)
		}
		pairs = append(pairs, "kind="+string(s.Kind))
	}
	if s.HasProgress && (s.State == ProgramWorking || s.State == ProgramBlocked) {
		p := min(max(s.Progress, 0), 100)
		pairs = append(pairs, "progress="+strconv.Itoa(p))
	}
	if s.App != "" {
		if !validProgramStatusSegment(s.App) {
			return "", fmt.Errorf("vaxis: invalid program status app %q", s.App)
		}
		pairs = append(pairs, "app="+s.App)
	}
	if title := programStatusText(s.Title, programStatusMaxTitle); title != "" {
		pairs = append(pairs, "title="+title)
	}
	if msg := programStatusText(s.Msg, programStatusMaxMsg); msg != "" {
		pairs = append(pairs, "msg="+msg)
	}
	return strings.Join(pairs, ":"), nil
}

// programStatusText strips control characters from s, truncates it to at most
// n bytes on a rune boundary and returns it base64 encoded.
func programStatusText(s string, n int) string {
	s = stripControls(s)
	if len(s) > n {
		for !utf8.RuneStart(s[n]) {
			n--
		}
		s = s[:n]
	}
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func validProgramStatusID(id string) bool {
	if len(id) > 128 {
		return false
	}
	segments := strings.Split(id, "/")
	if len(segments) > 8 {
		return false
	}
	for _, seg := range segments {
		if !validProgramStatusSegment(seg) {
			return false
		}
	}
	return true
}

func validProgramStatusSegment(s string) bool {
	if len(s) == 0 || len(s) > 32 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '_', c == '.', c == '+', c == '-':
		default:
			return false
		}
	}
	return true
}
