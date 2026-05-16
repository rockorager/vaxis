package term

func (vt *Model) c0(r rune) {
	switch r {
	case 0x05:
		vt.enquiry()
	case 0x07:
		vt.postEvent(EventBell{})
	case 0x08:
		vt.bs()
	case 0x09:
		vt.ht()
	case 0x0A:
		vt.lf()
	case 0x0B:
		vt.vt()
	case 0x0C:
		vt.ff()
	case 0x0D:
		vt.cr()
	case 0x0E:
		vt.charsets.selected = g1
	case 0x0F:
		vt.charsets.selected = g0
	}
}

func (vt *Model) enquiry() {
	vt.enqueueReplyString(vt.EnquiryResponse)
}

// Backspace 0x08
func (vt *Model) bs() {
	vt.cub(1)
}

// Horizontal tab 0x09
func (vt *Model) ht() {
	vt.cht(1)
}

// Linefeed 0x0A
func (vt *Model) lf() {
	vt.ind()
	vt.markSemanticContinuation()

	if !vt.mode.lnm {
		return
	}
	vt.cr()
}

// Vertical tabulation 0x0B
func (vt *Model) vt() {
	vt.lf()
}

// Form feed 0x0C
func (vt *Model) ff() {
	vt.lf()
}

// Carriage return 0x0D
func (vt *Model) cr() {
	vt.resetPendingWrap()
	if vt.mode.decom || vt.cursor.col >= vt.margin.left {
		vt.cursor.col = vt.margin.left
		return
	}
	vt.cursor.col = 0
}
