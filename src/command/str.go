package command

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
