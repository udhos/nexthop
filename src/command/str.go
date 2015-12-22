package command

// line: "interf  XXXX   descrip   YYY ZZZ WWW"
//              ^^    ^^^       ^^^
//              1     2         3
//
// IndexByte(s, ' ', 3): find 3rd sequence of ' ' in s

func IndexByte(s string, sep byte, n int) int {

	findSep := false
	found := 0
	for i := 0; i < len(s); i++ {
		if findSep {
			if s[i] == sep {
				found++
				if found == n {
					return i
				}
				findSep = false
			}
		} else {
			if s[i] != sep {
				findSep = true
			}
		}
	}

	return -1
}
