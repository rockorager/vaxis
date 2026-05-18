package term

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
// Form feed 0x0C
// Carriage return 0x0D
func (vt *Model) cr() {
	vt.resetPendingWrap()
	if vt.mode.decom || vt.cursor.col >= vt.margin.left {
		vt.cursor.col = vt.margin.left
		return
	}
	vt.cursor.col = 0
}
