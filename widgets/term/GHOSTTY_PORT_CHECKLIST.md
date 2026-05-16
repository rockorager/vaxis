# Ghostty Terminal Port Checklist

Goal: make `widgets/term` behaviorally match Ghostty's terminal state model for
core VT/xterm semantics, while staying idiomatic Go and allocation-conscious.

This checklist is the driver for the remaining work. Do not close the port based
only on broad test success; close it only when every section below has evidence
from code inspection, targeted behavior tests, and benchmarks where applicable.

## Source Map

Primary Ghostty sources to compare against:

- `../ghostty/src/terminal/Parser.zig`
- `../ghostty/src/terminal/stream.zig`
- `../ghostty/src/terminal/stream_terminal.zig`
- `../ghostty/src/terminal/Terminal.zig`
- `../ghostty/src/terminal/modes.zig`
- `../ghostty/src/terminal/dcs.zig`
- `../ghostty/src/terminal/osc.zig`
- `../ghostty/src/terminal/osc/parsers/*.zig`
- `../ghostty/src/terminal/charsets.zig`
- `../ghostty/src/terminal/kitty.zig`
- `../ghostty/src/terminal/device_attributes.zig`
- `../ghostty/src/terminal/device_status.zig`

Primary vaxis surfaces:

- `ansi/parser.go`, `ansi/parser_test.go`
- `widgets/term/action.go`
- `widgets/term/c0.go`, `esc.go`, `csi.go`, `sgr.go`, `osc.go`, `mode.go`
- `widgets/term/screen.go`, `term.go`
- `widgets/term/key.go`, `mouse.go`, `kitty_keyboard.go`
- `widgets/term/*_test.go`, `widgets/term/*_bench_test.go`

## Status Legend

- `done`: implemented and has targeted tests.
- `partial`: implemented or tested in part; needs Ghostty-source audit or more tests.
- `missing`: known absent behavior.
- `audit`: likely implemented, but not yet systematically checked against Ghostty.

## Parser And Stream Dispatch

- `done` Parser input/output modes: output mode avoids ESC key timers; input mode still supports ambiguous ESC and Kitty keyboard input.
- `done` ASCII allocation reduction: parser fast path avoids allocating for common ASCII prints.
- `done` C1 controls: CSI/OSC/DCS/APC/ST/NEL are parsed in output mode.
- `done` CSI parameter overflow: output CSI with too many params is dropped like Ghostty.
- `done` DCS parameter overflow: DCS is dropped like Ghostty.
- `done` CSI colon separator rule: output mode only dispatches colon-separated params for SGR final `m`; input mode remains permissive for Kitty keyboard.
- `done` Empty string terminators: empty OSC/DCS/APC terminated by `ESC \` consumes ST.
- `done` Parser test parity: every Ghostty `Parser.zig` test is mirrored in `ansi/parser_test.go` or covered by a semantic handler test below.
- `done` Unsupported parser paths: SOS/PM are explicit no-ops; APC is surfaced as `EventAPC`.

### Parser.zig Test Parity Matrix

Ghostty parser tests are action-level tests. Vaxis splits parsing from semantic
OSC handling, so OSC command assertions are considered covered only when there
is both raw parser coverage and `widgets/term` handler coverage.

| Ghostty `Parser.zig` test | Vaxis evidence | Status | Next action |
| --- | --- | --- | --- |
| anonymous C1 PM/ST, print, execute | `TestSOSPMControlStringsAreIgnored` covers C1 and 7-bit PM/SOS ignore, then print/C0 execute | done | None |
| `esc: ESC ( B` | `TestEscapeIntermediate/ESC ( B` | done | None |
| `csi: ESC [ H` | `TestCSI/CSI Entry + dispatch` | done | None |
| `csi: ESC [ 1 ; 4 H` | `TestCSI/CSI Param with multiple`, action cursor tests | done | None |
| `csi: SGR ESC [ 38 : 2 m` | `TestCSISGRColonParameterVariants/foreground mode` | done | None |
| `csi: SGR colon followed by semicolon` | `TestCSISGRColonParameterVariants`, parser reset covered by later CSI tests | done | None |
| `csi: SGR mixed colon and semicolon` | `TestCSISGRColonParameterVariants/indexed foreground and background` | done | None |
| `csi: SGR ESC [ 48 : 2 m` | `TestCSISGRColonParameterVariants/direct background` | done | None |
| `csi: SGR ESC [4:3m colon` | `TestCSISGRColonParameterVariants/underline style` | done | None |
| `csi: SGR with many blank and colon` | `TestCSISGRColonParameterVariants/blank color space` | done | None |
| `csi: SGR mixed colon and semicolon with blank` | `TestCSISGRColonParameterVariants/mixed blank and semicolon` | done | None |
| `csi: SGR mixed colon and semicolon setting underline, bg, fg` | `TestCSIWithManySGRParameters` | done | None |
| `csi: colon for non-m final` | `TestCSIColonParametersWithNonSGRFinalAreIgnored` | done | None |
| `csi: request mode decrqm` | `TestCSIGhosttyParserShapes/DECRQM`, higher-layer mode tests | done | None |
| `csi: change cursor` | `TestCSIGhosttyParserShapes/cursor style`, higher-layer cursor-style tests | done | None |
| `osc: change window title` | `TestOSC` raw OSC, `TestOSCTitleEvent` semantic handler | done | None |
| `osc: change window title (end in esc)` | `TestOSC/OSC 8 ...` covers ST payload, `TestOSCTitleEvent` covers title | done | None |
| `osc: 112 incomplete sequence` | `TestOSCDynamicColorSetAndReset` covers reset; raw parser emits raw OSC by design | done | None |
| `osc: 104 empty` | `TestOSC104WithoutParametersResetsAllPaletteColors` | done | None |
| `csi: too many params` | `TestCSIWithTooManyParametersIsIgnored` | done | None |
| `csi: sgr with up to our max parameters` | `TestCSIWithMaxParametersDispatches` | done | None |
| `csi: sgr beyond our max drops it` | `TestCSIWithTooManyParametersIsIgnored` | done | None |
| `dcs: XTGETTCAP` | `TestDCSGhosttyParserShapes/XTGETTCAP`, DCS handler tests | done | None |
| `dcs: params` | `TestDCSGhosttyParserShapes/params`, DCS handler tests | done | None |
| `dcs: too many params` | `TestDCSWithTooManyParametersIsIgnored` | done | None |

## Action Routing Matrix

Compare Ghostty `stream.zig` `Action` keys against vaxis routing in `action.go`,
`c0.go`, `esc.go`, `csi.go`, `osc.go`, `mode.go`, `sgr.go`, and `dcs` handling.

### Terminal.zig Test-Family Parity Matrix

Ghostty has 373 `Terminal.zig` tests. This matrix groups those tests by
behavioral family so the port can move family-by-family without losing the
source-map back to Ghostty's test names.

| Ghostty `Terminal.zig` family | Representative Ghostty tests | Vaxis evidence | Status | Next action |
| --- | --- | --- | --- | --- |
| Basic print, wrap, scroll, dirty rows, style-per-cell | input without controls, basic wraparound, forces scroll, unique style per cell, glitch text, long line | wrap/scrollback/resize/style tests, viewport tests, terminal action benchmarks; Ghostty dirty-cell checks map to vaxis's coarser redraw model | done | None |
| Wide characters and cell-boundary repair | print wide char, wide at edge, wide in single-width terminal, overwrite wide/spacer tail | wide/edit/resize tests across print, erase, delete, insert, resize; single-width policy is explicit in wrap tests | done | None |
| Grapheme clusters, VS15/VS16, emoji modifiers, ZWJ, combining marks | multicodepoint grapheme mode 2027, VS16 disabled/enabled/repeated, disabled-mode cluster splitting, invalid VS filtering including ZWJ/combining cases, Devanagari wide/wrap/bottom scroll, overwrite grapheme head/tail | `combining_test.go`, grapheme/uucode tests, DEC 2027 mode coverage; Ghostty per-cell dirty/refcount checks map to vaxis's coarser invalidation and inline cell storage | done | None |
| Charset designation and invocation | print charset, charset outside ASCII, locking/single shift, save/restore/reset, unsupported slots/finals ignored like Ghostty | `charset_test.go`, `action_test.go` charset tests | done | None |
| Right/left margins and wraparound | soft wrap, disabled wraparound with wide char/grapheme, right margin wrap/outside, mode toggles, margin boundary metadata | margin tests, wrap tests, DECAWM mode tests | done | None |
| Hyperlinks and row metadata movement | print with hyperlink, overwrite/change/end hyperlink, wide at edge with hyperlink | OSC 8 tests, resize hyperlink tests, scroll/edit/scroll-region hyperlink tests | done | None |
| C0 movement and tabs | LF/CR, LF mode, pending-wrap reset, CR with origin/LR margins, BS, HT/CBT with margins/origin and tab reset/resize | C0 tests, tab tests, LNM input/output tests | done | None |
| Cursor positioning and margins | cursorPos reset wrap/off screen/origin/LR, set top-bottom/LR margins invalid ranges, ambiguous `CSI s`, origin-mode home side effects | cursor and margin tests | done | None |
| Insert/delete lines and CSI scroll | insertLines variants, scrollUp/Down variants, scrollback creation/max zero, LR margins, cursor preservation, pending-wrap preservation/reset, wide boundary repair, hyperlink movement | edit tests, scroll tests, viewport/scrollback tests | done | None |
| Erase chars/line/display and protection | eraseChars/Line/Display with wide, SGR bg, ISO/DEC protected, DECSEL/DECSED force-protected forms, scroll complete, sixel clear | erase/protected tests, sixel erase tests | done | None |
| IND/RI/index movement | reverseIndex/index top/bottom, margins, scrollback, SGR bg, hyperlinks, cursor preservation | ESC/index/scroll tests | done | None |
| Insert blanks, insert mode, delete chars | insertBlanks, insert mode, deleteChars with wide/graphemes/hyperlinks/margins and Ghostty-style structural ops that ignore protected cells | insert/delete/insert-mode tests | done | None |
| Save/restore cursor and style state | saveCursor position/pending-wrap/origin/resize/protected pen/hyperlink; DECSC/DECRC intentionally do not restore DECAWM like Ghostty | save/restore tests, charset tests, resize saved-cursor tests | done | None |
| Style and DECALN | default/bold style, style GC, styled rows, DECALN reset margins/color/graphemes/protection | SGR tests, style tests, DECALN tests | done | None |
| REP and print attributes | printRepeat simple/wrap/no previous/wide/grapheme/charset, printAttributes | REP and SGR/DECRQSS tests | done | None |
| Semantic prompt OSC 133 | prompt marks, continuations, input/output mode, OSC133C/A options, cursor at prompt | semantic prompt tests | done | None |
| Full reset | pen, hyperlink, saved cursor, origin, status display, alt-screen Kitty state, default modes, OSC color overrides, previous character, scroll offset, charsets, tabs, sixels | RIS/reset tests, status display, Kitty keyboard, OSC color, DCS/sixel, charset, REP tests | done | None |
| Resize and reflow | less cols wide print, margins, wrap on/off, high style, saved cursor, pending-wrap | extensive `resize_test.go`, viewport tests, scrollback tests, semantic-prompt and hyperlink resize tests | done | None |
| DECCOLM | gated by mode 40, unset, pending-wrap reset, SGR bg, scroll-region reset, host-size independence, save/restore of mode 40 gate | `deccolm_test.go` | done | None |
| Alt screen modes | mode 47 retains alt content, 1047 clears alt on exit, 1049 saves/restores primary cursor and clears alt on entry, cursor state copies both directions, viewport offset clears | `alt_screen_test.go`, `resize_test.go`, Kitty keyboard alt-screen tests | done | None |

Intentional Go-native difference: Ghostty's `print breaks valid grapheme cluster
with Prepend + ASCII for speed` test documents a deliberate deviation from UAX
#29. Vaxis keeps the shared `uucode` parser's UAX #29 clustering for this case
instead of copying Ghostty's optimization unless benchmarks show a need to
special-case it.

### `stream.zig` Action Coverage

This table records every Ghostty stream action and the current vaxis route.
`done` means there is an implemented route and targeted vaxis tests; `partial`
means behavior exists but still needs Ghostty semantic audit or additional
tests; `missing` means no equivalent route exists yet.

| Ghostty action | Vaxis route/evidence | Status | Next action |
| --- | --- | --- | --- |
| `print` | `ansi.Print` -> `vt.print`; print/edit/combining tests | done | None |
| `print_repeat` | `CSI b` -> `vt.rep`; REP tests | done | None |
| `bell` | `C0 BEL` -> `EventBell`; C0 tests | done | None |
| `backspace` | `C0 BS` -> `vt.bs`; C0 tests | done | None |
| `horizontal_tab` | `C0 HT`, `CSI I` -> `vt.ht`/`vt.cht`; tab tests | done | None |
| `horizontal_tab_back` | `CSI Z` -> `vt.cbt`; tab tests | done | None |
| `linefeed` | `C0 LF/VT/FF` -> `vt.lf`; C0 tests | done | None |
| `carriage_return` | `C0 CR` -> `vt.cr`; C0 tests | done | None |
| `enquiry` | `C0 ENQ` -> `vt.enquiry`; C0 tests | done | None |
| `invoke_charset` | SO/SI, SS2/SS3, LS2/LS3, LS1R/LS2R/LS3R routes; charset tests cover single-use restore, locking shifts, GR save/restore | done | None |
| `configure_charset` | ESC `(`,`)`,`*`,`+` for ASCII/British/DEC special; tests cover Ghostty's current rejection of unsupported `-`, `.`, `/` slots and unknown finals | done | None |
| `cursor_up` | `CSI A/k` -> `vt.cuu`; tests cover default/zero, invalid parameter count, margins, pending-wrap reset | done | None |
| `cursor_down` | `CSI B` -> `vt.cud`; tests cover default/zero, invalid parameter count, margins, pending-wrap reset | done | None |
| `cursor_left` | `CSI D/j` -> `vt.cub`; tests cover default/zero, invalid parameter count, pending-wrap reset, reverse-wrap and extended reverse-wrap | done | None |
| `cursor_right` | `CSI C` -> `vt.cuf`; tests cover default/zero, invalid parameter count, margins, pending-wrap reset | done | None |
| `cursor_col` | `CSI G/\`` -> `vt.cha`/`vt.hpa`; tests cover default/zero, invalid parameter count, origin/LR margins | done | None |
| `cursor_row` | `CSI d` -> `vt.vpa`; tests cover default/zero, invalid parameter count, origin/TB margins | done | None |
| `cursor_col_relative` | `CSI a` -> `vt.hpr`; tests cover default/zero, invalid parameter count, origin/LR margins and clamping | done | None |
| `cursor_row_relative` | `CSI e` -> `vt.vpr`; tests cover default/zero, invalid parameter count, origin/TB margins and clamping | done | None |
| `cursor_pos` | `CSI H/f` -> `vt.cup`; tests cover default/zero, invalid parameter count, screen clamping, origin mode, LR/TB margins, pending-wrap reset | done | None |
| `cursor_style` | `CSI SP q` -> `vt.decscusr`; cursor-style tests | done | None |
| `erase_display_below` | `CSI J 0`, `CSI ? J 0` -> `vt.ed`; tests cover wide splits, background SGR, ISO/DEC protected behavior, DECSED force protection | done | None |
| `erase_display_above` | `CSI J 1`, `CSI ? J 1` -> `vt.ed`; tests cover wide splits, background SGR, ISO/DEC protected behavior, DECSED force protection | done | None |
| `erase_display_complete` | `CSI J 2`, `CSI ? J 2` -> `vt.ed`; tests cover cursor preservation, semantic-prompt scroll clear, protected complete, background SGR, sixel clear | done | None |
| `erase_display_scrollback` | `CSI J 3` -> `vt.ed`; clears active screen scrollback only and clamps viewport; scrollback tests | done | None |
| `erase_display_scroll_complete` | `CSI J 22` -> `vt.ed`; clears active display plus scrollback, resets pending wrap, clears sixel graphics | done | None |
| `erase_line_right` | `CSI K 0`, `CSI ? K 0` -> `vt.el`; tests cover pending/soft-wrap reset, wide split, background SGR, ISO/DEC protected behavior, DECSEL force protection | done | None |
| `erase_line_left` | `CSI K 1`, `CSI ? K 1` -> `vt.el`; tests cover pending-wrap reset, wide split, background SGR, ISO/DEC protected behavior, DECSEL force protection | done | None |
| `erase_line_complete` | `CSI K 2`, `CSI ? K 2` -> `vt.el`; tests cover background SGR, ISO/DEC protected behavior, DECSEL force protection | done | None |
| `erase_line_right_unless_pending_wrap` | Ghostty defines the enum but rejects `EL 4` before routing; vaxis rejects `EL 4` with `TestEraseLineIgnoresInvalidParameters/right unless pending wrap` | done | None |
| `delete_chars` | `CSI P` -> `vt.dch`; tests cover defaults, zero/no-op, margins, pending-wrap reset, wide splits, graphemes, hyperlinks, background fill; structural op intentionally ignores protection like Ghostty | done | None |
| `erase_chars` | `CSI X` -> `vt.ech`; tests cover default/zero, wide splits, background SGR, hyperlinks, ISO/DEC protection | done | None |
| `insert_lines` | `CSI L` -> `vt.il`; tests cover defaults, zero/no-op, margins, pending-wrap reset, wide boundary repair, row metadata, background fill | done | None |
| `insert_blanks` | `CSI @` -> `vt.ich`; tests cover defaults, zero/pending-wrap, margins, wide/grapheme splits, hyperlinks, background fill; structural op intentionally ignores protection like Ghostty | done | None |
| `delete_lines` | `CSI M` -> `vt.dl`; tests cover defaults, zero/no-op, margins, pending-wrap reset, wide boundary repair, row metadata, background fill | done | None |
| `scroll_up` | `CSI S` -> `vt.scrollUp`; tests cover defaults, zero/no-op, cursor preservation, pending-wrap preservation, LR/TB margins, hyperlink movement/clear, primary scrollback creation and disabled scrollback | done | None |
| `scroll_down` | `CSI T` -> `vt.scrollDown`; tests cover defaults, zero/no-op, cursor preservation, pending-wrap preservation, LR/TB margins, hyperlink movement/clear, outside-region no-op | done | None |
| `tab_clear_current` | `CSI g 0`, `CSI W 2` -> `vt.tbc`/`vt.ctc`; tests cover explicit-parameter requirement, current-stop clear, invalid/private ignored | done | None |
| `tab_clear_all` | `CSI g 3`, `CSI W 5` -> `vt.tbc`/`vt.ctc`; tests cover clearing all stops and overflowing-parameter ignore | done | None |
| `tab_set` | `ESC H`, `CSI W`/`CSI 0 W` -> `vt.hts`/`vt.ctc`; tests cover duplicate-safe set and forward/back tab movement with margins/origin | done | None |
| `tab_reset` | `CSI ? 5 W` -> `vt.ctc`; tests cover default tab reset and width-resize tab reset | done | None |
| `index` | `ESC D` -> `vt.ind`; ESC/scroll tests | done | None |
| `next_line` | `ESC E` -> `vt.nel`; ESC tests | done | None |
| `reverse_index` | `ESC M` -> `vt.ri`; ESC/scroll tests | done | None |
| `full_reset` | `ESC c` -> `vt.ris`; clears primary/alternate screens, scrollback, cursor/saved cursor, modes/saved modes, margins, tabs, charsets, title/pwd, status display, OSC colors, prompt state, previous char, sixels, and Kitty keyboard state | done | None |
| `set_mode` | `CSI h`, `CSI ? h` -> `sm`/`decset`; all Ghostty ANSI/DEC `modes.zig` rows mapped, defaults and side effects tested | done | None |
| `reset_mode` | `CSI l`, `CSI ? l` -> `rm`/`decrst`; all Ghostty ANSI/DEC `modes.zig` rows mapped, defaults and side effects tested | done | None |
| `save_mode` | `CSI ? s` -> `saveMode`; all Ghostty DEC modes save current value, unknown modes ignored, mode report/save-restore tests cover matrix | done | None |
| `restore_mode` | `CSI ? r` -> `restoreMode`; restore routes through `setDECMode` so side effects match Ghostty's restore-then-set flow | done | None |
| `request_mode` | `DECRQM`/`ANSI RQM` -> `decrqm`; all Ghostty ANSI/DEC `modes.zig` rows plus unknown reports covered | done | None |
| `request_mode_unknown` | unknown `DECRQM` replies unsupported; mode report tests | done | None |
| `top_and_bottom_margin` | `CSI r` -> `vt.decstbm`; tests cover 0/defaults, clamping, equal/invalid ranges ignored, too many params ignored, and origin-mode cursor home | done | None |
| `left_and_right_margin` | `CSI s` under `DECLRMM` -> `vt.decslrm`; tests cover DEC 69 gating, 0/defaults, clamping, equal/invalid ranges ignored, too many params ignored, and origin-mode cursor home | done | None |
| `left_and_right_margin_ambiguous` | `CSI s` save-cursor or reset LR margins; margin tests | done | None |
| `save_cursor` | `ESC 7`, `CSI s` -> `vt.decsc`; saves position, SGR pen, protected pen, pending wrap, origin mode, and charsets, without saving DECAWM | done | None |
| `restore_cursor` | `ESC 8`, `CSI u` -> `vt.decrc`; restores saved payload, defaults when unsaved, clamps after resize, preserves hyperlink/semantic state, and does not restore DECAWM | done | None |
| `modify_key_format` | `CSI > m`, `CSI > n` -> modifyOtherKeys; tests | done | None |
| `mouse_shift_capture` | `CSI > s` -> `vt.xtshiftescape`; mouse tests | done | None |
| `protected_mode_off` | `ESC W`, `CSI \" q 0/2` -> protected off; preserves most-recent ISO/DEC mode like Ghostty; tests cover invalid params and saved cursor | done | None |
| `protected_mode_iso` | `ESC V` -> protected ISO; normal ECH/EL/ED respect protected cells while ISO is most recent | done | None |
| `protected_mode_dec` | `CSI \" q 1` -> protected DEC; normal erases ignore protected cells when DEC is most recent, DECSEL/DECSED force protection | done | None |
| `size_report` | `CSI t 14/16/18` -> `xtwinops`; xtwinops tests | done | None |
| `title_push` | `CSI t 22` parsed as explicit no-op; xtwinops tests | done | None |
| `title_pop` | `CSI t 23` parsed as explicit no-op; xtwinops tests | done | None |
| `xtversion` | `CSI > q` -> `vt.xtversion`; device tests | done | None |
| `device_attributes` | DA1/DA2/DA3 -> reply handlers; DA2 matches Ghostty app firmware `10`; DA1 intentionally keeps vaxis sixel feature instead of Ghostty clipboard feature; DA3 supported like Ghostty stream library | done | None |
| `device_status` | DSR 5/6 and private color-scheme 996 match Ghostty supported request set; public/private invalid forms ignored; tests cover operating, cursor, origin-relative cursor, dark/light/unknown color-scheme replies | done | None |
| `kitty_keyboard_query` | `CSI ? u`; host-gated by `WithVaxis`/`WithKittyKeyboard`; tests cover disabled query, TERM gating, current flags, per-screen state, associated text/report-all input behavior | done | None |
| `kitty_keyboard_push` | `CSI > u`; host-gated fixed 8-entry Ghostty-style ring stack; tests cover defaults, invalid flags, overflow, per-screen state | done | None |
| `kitty_keyboard_pop` | `CSI < u`; host-gated pop defaults to one and large counts reset stack like Ghostty; tests cover explicit/default/large pop | done | None |
| `kitty_keyboard_set` | `CSI = u`; host-gated set action; tests cover replacement and invalid flags | done | None |
| `kitty_keyboard_set_or` | `CSI = u` submode support; OR action covered against Ghostty flag bits | done | None |
| `kitty_keyboard_set_not` | `CSI = u` submode support; NOT action covered against Ghostty flag bits | done | None |
| `dcs_hook`/`dcs_put`/`dcs_unhook` | Vaxis parser emits complete `ansi.DCS`; XTGETTCAP/DECRQSS match Ghostty app-side handling; unsupported DCS ignored; sixel intentionally retained as vaxis terminal-widget behavior | done | None |
| `apc_start`/`apc_put`/`apc_end` | Vaxis parser emits complete `ansi.APC` event | done | None |
| `end_hyperlink` | OSC 8 empty URL -> clear hyperlink; OSC8 tests | done | None |
| `active_status_display` | `CSI $ }` -> `vt.decsasd`; status-display tests | done | None |
| `decaln` | `ESC # 8` -> `vt.decaln`; ESC tests | done | None |
| `window_title` | OSC 0/2 -> title state/event; OSC tests | done | None |
| `report_pwd` | OSC 7/9;9/iTerm2 CurrentDir -> event; OSC tests | done | None |
| `show_desktop_notification` | OSC 9 body notifications and OSC 777 rxvt `notify;title;body` -> `EventNotify`; ConEmu subcommands routed to pwd/progress/prompt or no-op before notification fallback | done | None |
| `progress_report` | OSC 9 ConEmu progress -> event; OSC tests | done | None |
| `start_hyperlink` | OSC 8 -> cursor hyperlink; OSC8 tests | done | None |
| `clipboard_contents` | OSC 52/iTerm2 Copy writes clipboard only when a host `Vaxis` is attached; invalid base64/malformed data ignored; read queries are explicit no-ops; Kitty OSC 5522 explicit no-op | done | None |
| `mouse_shape` | OSC 22 -> mouse shape event/state; OSC tests | done | None |
| `set_attribute` | SGR -> `vt.sgr`; tests cover reset, boolean attrs/reset, underline styles/colors, 8/bright/256/RGB fg/bg, overline, colon/semicolon parser quirks, invalid/short groups ignored | done | None |
| `kitty_color_report` | OSC 21 Kitty color protocol; tests cover palette/dynamic set/reset, query no-op, invalid entry skip, named/rgb/rgbi parsing | done | None |
| `color_operation` | OSC 4/5/10-19/104/105/110-119; tests cover palette set/reset/query no-op, dynamic range, resets, invalid/special colors, named/rgb/rgbi parsing | done | None |
| `semantic_prompt` | OSC 133/9;12/iTerm2 marks rows/cells; tests cover prompt/input/output transitions, A/P/B/C/I/L forms, redraw/click options, continuation rows, erase/reset/save-restore, alternate screen behavior | done | None |

- `done` C0 core: bell, BS, HT, LF, CR, ENQ.
- `done` ESC core: DECSC/DECRC, IND/NEL/RI, HTS, RIS, DECALN, keypad mode, C1 7-bit forms.
- `done` ESC charsets: G0-G3, GL/GR, SS2/SS3, DEC special, ASCII, British, non-ASCII fallback, save/restore/reset, and Ghostty's current unsupported `-`, `.`, `/` designators are covered.
- `done` CSI cursor movement: CUU/CUD/CUF/CUB/CNL/CPL/CHA/CUP/HPA/VPA/HPR/VPR plus aliases are covered with Ghostty defaults, zero handling, invalid parameter rejection, margin/origin interactions, pending-wrap reset, reverse-wrap, and clamping.
- `done` CSI editing: ICH/IL/DL/DCH/ECH/EL/ED are covered with margins, wide cells, protected cells, background SGR, hyperlink movement, and Ghostty wide-boundary cases.
- `done` CSI scrolling: SU/SD, IND/RI and top/bottom/left/right margins are covered with cursor preservation, scrollback creation, pending-wrap preservation, hyperlinks, and background SGR.
- `done` CSI tabs: HT/CBT/TBC/CTC, private reset, explicit parameter handling, default interval, and width-resize reset covered against Ghostty `Tabstops.zig` behavior.
- `done` CSI SGR: basic attributes, underline styles/colors, RGB/indexed colors, overline, colon forms, unknown/short groups, and full Ghostty `sgr.zig` parser behavior are covered for vaxis-supported style fields.
- `done` Device reports: DA1/DA2/DA3, DSR 5/6/`?996`, DECRQM, XTVERSION, XTWINOPS covered. DA1 intentionally advertises sixel for vaxis compatibility, while Ghostty app advertises clipboard when available; DA2 follows Ghostty app firmware `10`.
- `done` DCS: XTGETTCAP, DECRQSS, unsupported-command no-ops, invalid long DECRQSS handling, and parser shapes are covered. Vaxis intentionally retains sixel decoding/placement even though Ghostty's terminal core has no screen-mutating DCS.
- `done` OSC: title, OSC 7/8/9/10-19/21/22/52/104/110-119/133/1337, unsupported no-ops, invalid data, color reset/query behavior, and semantic prompt side effects are covered.
- `done` Kitty keyboard: host gating, TERM support check, per-screen state, query, push/pop/set/set-or/set-not, stack overflow/large pop behavior, associated text, report-all, and event encoding are covered against Ghostty `kitty.zig`.
- `done` Mouse: X10, normal, button, any, UTF8, SGR, URXVT, pixel modes covered with modifier/button encoding, active-mode precedence, shape side effects, save/restore/report, and legacy-vs-SGR release behavior.

## Modes

Compare every `modes.zig` entry to `mode.go`, save/restore, DECRQM, and full reset.

### `modes.zig` Coverage

This table maps Ghostty's mode entries to vaxis mode storage and current
verification. All Ghostty mode values have corresponding vaxis state; remaining
work is mostly side-effect parity rather than missing storage.

| Ghostty mode | Value | Default | Vaxis state/route | Status | Next action |
| --- | ---: | --- | --- | --- | --- |
| `disable_keyboard` | ANSI 2 | reset | `kam`, `SM/RM`, `DECRQM`, opt-in `WithKeyboardActionMode` input suppression | done | None |
| `insert` | ANSI 4 | reset | `irm`, insert-mode print path, `DECRQM` | done | None |
| `send_receive_mode` | ANSI 12 | set | `srm`, `DECRQM` | done | None |
| `linefeed` | ANSI 20 | reset | `lnm`, output LF auto-CR, input CR-to-CRLF expansion, `DECRQM` | done | None |
| `cursor_keys` | DEC 1 | reset | `decckm`; normal/application encoding for Up/Down/Right/Left/Home/End, `DECRQM`, save/restore covered | done | None |
| `132_column` | DEC 3 | reset | `deccolm`, gated by `enableMode3`, resize to 80/132, erase display, home cursor, reset pending wrap | done | None |
| `slow_scroll` | DEC 4 | reset | `decsclm`, report-only/no visual effect | done | None |
| `reverse_colors` | DEC 5 | reset | `decscnm`, render-time reverse-video transform, `DECRQM` | done | None |
| `origin` | DEC 6 | reset | `decom`, homes cursor on set/reset, margin-relative CUP/HPA/VPA/HPR/VPR, CR/tabs, DSR, save/restore/full reset covered | done | None |
| `wraparound` | DEC 7 | set | `decawm`, pending-wrap handling, disabled wide/grapheme drop, save/restore/report/full reset, soft-wrap metadata preservation | done | None |
| `autorepeat` | DEC 8 | reset | `decarm`, report-only | done | None |
| `mouse_event_x10` | DEC 9 | reset | `mouseX10`; active event mode, basic press-only reporting, modifier suppression, coordinate limit, shape/report/save/restore covered | done | None |
| `cursor_blinking` | DEC 12 | reset | `cursorBlinking`, DECSCUSR integration | done | None |
| `cursor_visible` | DEC 25 | set | `dectcem`, cursor visibility | done | None |
| `enable_mode_3` | DEC 40 | reset | `enableMode3`, gates DECCOLM; report/save/restore/full reset and no-resize side effects covered | done | None |
| `reverse_wrap` | DEC 45 | reset | `reverseWrap`; cursor-left reverse wrap through soft-wrapped rows, margin limits, first-row behavior, report/save/restore covered | done | None |
| `alt_screen_legacy` | DEC 47 | reset | `smcup`, alt screen switch, content retention, cursor copy both directions, report/save/restore covered | done | None |
| `keypad_keys` | DEC 66 | reset | `deckpam`/`deckpnm` | done | None |
| `backarrow_key_mode` | DEC 67 | reset | `decbkm`; plain/Ctrl DECBKM inversion, Ghostty legacy modified-backspace table, Kitty/report-all behavior, `DECRQM`, save/restore covered | done | None |
| `enable_left_and_right_margin` | DEC 69 | reset | `declrmm`, gates DECSLRM, LR margins reset on disable, report/save/restore covered | done | None |
| `mouse_event_normal` | DEC 1000 | reset | `mouseButtons`; active event mode, press/release reporting, motion suppression, legacy release encoding, shape/report/save/restore covered | done | None |
| `mouse_event_button` | DEC 1002 | reset | `mouseDrag`; active event mode, drag-only motion reporting, no-button motion suppression, report/save/restore covered | done | None |
| `mouse_event_any` | DEC 1003 | reset | `mouseMotion`; active event mode, no-button motion reporting, event-mode precedence, report/save/restore covered | done | None |
| `focus_event` | DEC 1004 | reset | `focusEvents`, reports current focus state on enable | done | None |
| `mouse_format_utf8` | DEC 1005 | reset | `mouseUTF8`; active format, UTF-8 coordinate encoding, reset/report/save/restore covered | done | None |
| `mouse_format_sgr` | DEC 1006 | reset | `mouseSGR`; active format, press/release/motion/wheel/extended button/modifier encoding, reset/report/save/restore covered | done | None |
| `mouse_alternate_scroll` | DEC 1007 | set | `altScroll` | done | None |
| `mouse_format_urxvt` | DEC 1015 | reset | `mouseURXVT`; active format, modifier encoding, legacy release button-3 encoding, reset/report/save/restore covered | done | None |
| `mouse_format_sgr_pixels` | DEC 1016 | reset | `mouseSGRPixels`; active format, terminal-space pixel press/release encoding, reset/report/save/restore covered | done | None |
| `ignore_keypad_with_numlock` | DEC 1035 | set | `ignoreKeypadWithNumLock` | done | None |
| `alt_esc_prefix` | DEC 1036 | set | `altEscPrefix` | done | None |
| `alt_sends_escape` | DEC 1039 | reset | `altSendsEscape` | done | None |
| `reverse_wrap_extended` | DEC 1045 | reset | `reverseWrapExtended`; extended cursor-left wrap ignores soft-wrap metadata, wraps top-to-bottom, takes priority over DEC 45, report/save/restore covered | done | None |
| `alt_screen` | DEC 1047 | reset | `smcup`, alt switch, cursor copy both directions, clear-on-exit behavior, report/save/restore covered | done | None |
| `save_cursor` | DEC 1048 | reset | `saveCursor`, active-screen DECSC/DECRC side effect, pending-wrap restore, report/save/restore covered | done | None |
| `alt_screen_save_cursor_clear_enter` | DEC 1049 | reset | `smcup`, save primary cursor, switch alt, clear on entry, restore primary cursor on exit, repeated-enable behavior, report/save/restore covered | done | None |
| `bracketed_paste` | DEC 2004 | reset | `paste` | done | None |
| `synchronized_output` | DEC 2026 | reset | `synchronizedOutput`, resize reset, Ghostty-style safety timeout | done | None |
| `grapheme_cluster` | DEC 2027 | reset | `graphemeCluster`; parser clusters split when disabled, invalid VS15/VS16 filtered like Ghostty, wide grapheme wrap/scroll and overwrite head/tail covered; per-cell dirty/refcount Ghostty tests intentionally map to vaxis invalidation/storage model | done | None |
| `report_color_scheme` | DEC 2031 | reset | `colorScheme`; forced `CSI ? 996 n` query replies, mode-gated unsolicited theme updates, unknown-theme suppression, report/save/restore covered | done | None |
| `in_band_size_reports` | DEC 2048 | reset | `inBandSizeReports`, report on enable/resize | done | None |

- `done` Default modes: SRM, DECAWM, DECTCEM, alternate scroll, ignore-keypad, alt-esc-prefix.
- `done` ANSI modes: KAM, IRM, SRM, LNM implemented with report and targeted side-effect tests.
- `done` DEC modes: all Ghostty mode rows in this checklist have an implemented vaxis route plus targeted side-effect/report/save/restore coverage.
- `done` Audited mode side effects: origin mode home, DECCOLM clear/resize, DECLRMM reset/gating, alt screen cursor copy/clear rules.

## Screen Model

- `done` Flat cell storage with row metadata separate.
- `done` Primary scrollback and viewport are first-class and tested.
- `done` Primary/alternate screen storage and cursor state are separate.
- `done` Row metadata: wrapped, wrap continuation, semantic prompt, hyperlinks, protection, and sixel graphics are covered through print, edit, scroll, erase, resize, reset, and screen-switch tests.
- `done` Resize: same-width grow/shrink, scrollback retention, reflow, saved cursor remap, prompt redraw, hyperlink preservation, wide-boundary repair, margin reset, and synchronized-output reset are covered.
- `done` Sixel: original support is retained with DCS parsing, parameterized input, placement at cursor origin, RIS clearing, and ED 2/22 clearing covered. Ghostty's terminal core does not provide sixel screen mutation, so this is an intentional vaxis extension.

## Verification Requirements

Before considering the port complete:

- Build a test parity matrix for Ghostty `Parser.zig` tests.
- Build a test-family parity matrix for Ghostty `Terminal.zig` test names.
- For every `stream.zig` action, record one of: implemented+tested, explicit no-op+tested, documented gap.
- For every `modes.zig` entry, record set/reset/report/save/restore/full-reset behavior.
- Keep these gates green after each implementation slice:
  - `go test -count=1 -timeout=120s ./widgets/term`
  - `go test -count=1 ./...`
  - `git diff --check`
  - `go test -run '^$' -bench 'BenchmarkParser' -benchmem ./ansi`
  - `go test -run '^$' -bench 'BenchmarkScreenBuffer|BenchmarkTerminalActions|BenchmarkViewport' -benchmem ./widgets/term`

## Performance Requirements

- Preserve `0 B/op, 0 allocs/op` on common terminal action paths where practical.
- Do not reintroduce `ansi.Pool` or string pooling unless a benchmark proves a net win.
- Keep screen storage flat and avoid per-cell heap allocations in hot paths.
- Track parser benchmark regressions separately from terminal action regressions.

Latest verification snapshot, 2026-05-16:

- `go test -count=1 -timeout=120s ./widgets/term`: pass.
- `go test -count=1 ./...`: pass.
- `git diff --check`: pass.
- `go test -run '^$' -bench 'BenchmarkParser' -benchmem ./ansi`: pass; measured parser allocations remain on the parser event path (`control` 349166 B/op 3081 allocs/op, `csi` 332777 B/op 6153 allocs/op, `mixed` 517101 B/op 13833 allocs/op, `plain` 269286 B/op 11017 allocs/op).
- `go test -run '^$' -bench 'BenchmarkScreenBuffer|BenchmarkTerminalActions|BenchmarkViewport' -benchmem ./widgets/term`: pass; `BenchmarkTerminalActions/*`, `BenchmarkScreenBuffer` except scrollback backing storage, and `BenchmarkViewport/*` preserve 0 allocs/op.

## Work Order

1. Finish the parser parity matrix and close any remaining `Parser.zig` mismatches.
2. Build the `stream.zig` action routing matrix and close missing/ambiguous routes.
3. Build the `modes.zig` matrix and close mode side-effect/report/save/restore gaps.
4. Audit `Terminal.zig` test families in order: print/grapheme, cursor, margins/tabs, editing/erase, scroll/scrollback, save/restore, resize/reflow, alt screen.
5. Audit OSC/DCS/device reports using Ghostty parser files, preserving sixel support and explicit no-ops.
6. Audit key/mouse/Kitty behavior with host-support gating.
7. Run the completion audit against this checklist and the verification gates.
