package main

func sliceContainsStr(a []string, b string) bool {
	for _, c := range a {
		if c == b {
			return true
		}
	}
	return false
}

func sliceEqStr(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func sliceSubStr(a, b []string) []string {
	var result []string

	for _, i := range a {
		if !sliceContainsStr(b, i) {
			result = append(result, i)
		}
	}

	return result
}
