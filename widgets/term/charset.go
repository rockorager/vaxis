package term

type charset int

const (
	ascii charset = iota
	british
	decSpecialAndLineDrawing
)

type charsets struct {
	designations [4]charset
	selected     charsetDesignator
	saved        charsetDesignator
	singleShift  bool
}

type charsetDesignator int

const (
	g0 = iota
	g1
	g2
	g3
)

var britishTable = func() [256]rune {
	table := identityCharsetTable()
	table[0x23] = 0x00A3 // POUND SIGN
	return table
}()

var decSpecialTable = func() [256]rune {
	table := identityCharsetTable()
	table[0x5f] = 0x00A0 // NO-BREAK SPACE
	table[0x60] = 0x25C6 // BLACK DIAMOND
	table[0x61] = 0x2592 // MEDIUM SHADE
	table[0x62] = 0x2409 // SYMBOL FOR HORIZONTAL TABULATION
	table[0x63] = 0x240C // SYMBOL FOR FORM FEED
	table[0x64] = 0x240D // SYMBOL FOR CARRIAGE RETURN
	table[0x65] = 0x240A // SYMBOL FOR LINE FEED
	table[0x66] = 0x00B0 // DEGREE SIGN
	table[0x67] = 0x00B1 // PLUS-MINUS SIGN
	table[0x68] = 0x2424 // SYMBOL FOR NEWLINE
	table[0x69] = 0x240B // SYMBOL FOR VERTICAL TABULATION
	table[0x6a] = 0x2518 // BOX DRAWINGS LIGHT UP AND LEFT
	table[0x6b] = 0x2510 // BOX DRAWINGS LIGHT DOWN AND LEFT
	table[0x6c] = 0x250C // BOX DRAWINGS LIGHT DOWN AND RIGHT
	table[0x6d] = 0x2514 // BOX DRAWINGS LIGHT UP AND RIGHT
	table[0x6e] = 0x253C // BOX DRAWINGS LIGHT VERTICAL AND HORIZONTAL
	table[0x6f] = 0x23BA // HORIZONTAL SCAN LINE-1
	table[0x70] = 0x23BB // HORIZONTAL SCAN LINE-3
	table[0x71] = 0x2500 // BOX DRAWINGS LIGHT HORIZONTAL
	table[0x72] = 0x23BC // HORIZONTAL SCAN LINE-7
	table[0x73] = 0x23BD // HORIZONTAL SCAN LINE-9
	table[0x74] = 0x251C // BOX DRAWINGS LIGHT VERTICAL AND RIGHT
	table[0x75] = 0x2524 // BOX DRAWINGS LIGHT VERTICAL AND LEFT
	table[0x76] = 0x2534 // BOX DRAWINGS LIGHT UP AND HORIZONTAL
	table[0x77] = 0x252C // BOX DRAWINGS LIGHT DOWN AND HORIZONTAL
	table[0x78] = 0x2502 // BOX DRAWINGS LIGHT VERTICAL
	table[0x79] = 0x2264 // LESS-THAN OR EQUAL TO
	table[0x7a] = 0x2265 // GREATER-THAN OR EQUAL TO
	table[0x7b] = 0x03C0 // GREEK SMALL LETTER PI
	table[0x7c] = 0x2260 // NOT EQUAL TO
	table[0x7d] = 0x00A3 // POUND SIGN
	table[0x7e] = 0x00B7 // MIDDLE DOT
	return table
}()

func identityCharsetTable() [256]rune {
	var table [256]rune
	for i := range table {
		table[i] = rune(i)
	}
	return table
}

func applyCharset(set charset, r rune) rune {
	switch set {
	case british:
		if r > 0xFF {
			return ' '
		}
		return britishTable[byte(r)]
	case decSpecialAndLineDrawing:
		if r > 0xFF {
			return ' '
		}
		return decSpecialTable[byte(r)]
	default:
		return r
	}
}
