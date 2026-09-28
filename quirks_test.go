package vaxis

import "testing"

func TestTmuxUnicodeQuirk(t *testing.T) {
	for _, env := range []string{
		"VAXIS_FORCE_WCWIDTH", "VAXIS_FORCE_UNICODE", "VAXIS_FORCE_LEGACY_SGR",
	} {
		t.Setenv(env, "")
	}

	for _, tt := range []struct {
		id   string
		want bool
	}{
		{"tmux 1.9a", false},
		{"tmux 2.9a", false},
		{"tmux 2.10", false},
		{"tmux 3.3", false},
		{"tmux 3.3a", false},
		{"tmux 3.4", true},
		{"tmux 3.4a", true},
		{"tmux 3.5a", true},
		{"tmux 3.10", true},
		{"tmux 4.0", true},
		{"tmux 10.0", true},
		{"tmux 3.3-rc", false},
		{"tmux 3.4-rc", true},
		{"tmux 3.8-rc2", true},
		{"tmux next-3.3", false},
		{"tmux next-3.4", true},
		{"tmux next-3.9", true},
		{"tmux ", false},
		{"tmux next-", false},
		{"tmux unknown", false},
		{"tmux 4", false},
		{"tmux 4.", false},
		{"tmux 3.x", false},
		{"other 3.4", false},
		{"", false},
	} {
		t.Run(tt.id, func(t *testing.T) {
			vx := &Vaxis{termID: terminalID(tt.id)}
			vx.applyQuirks()
			if vx.caps.unicodeCore != tt.want {
				t.Errorf("unicodeCore = %v, want %v", vx.caps.unicodeCore, tt.want)
			}
		})
	}

	t.Run("preserve detected support", func(t *testing.T) {
		vx := &Vaxis{termID: "tmux 3.3a", caps: capabilities{unicodeCore: true}}
		vx.applyQuirks()
		if !vx.caps.unicodeCore {
			t.Error("quirk disabled detected unicode support")
		}
	})

	t.Run("wcwidth override", func(t *testing.T) {
		t.Setenv("VAXIS_FORCE_WCWIDTH", "1")
		vx := &Vaxis{termID: "tmux 3.5a"}
		vx.applyQuirks()
		if vx.caps.unicodeCore {
			t.Error("quirk ignored VAXIS_FORCE_WCWIDTH")
		}
	})
}
