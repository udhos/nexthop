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

func longestCommonPrefix(set []string) string {

	if set == nil {
		return ""
	}

	if len(set) < 1 {
		return ""
	}

	first := set[0]

	// scan first string
	for i, c := range first {

		// scan other strings
		for j := 1; j < len(set); j++ {
			s := set[j]

			if i >= len(s) {
				return first[:i]
			}

			if byte(c) != s[i] {
				return first[:i]
			}
		}
	}

	return first
}
