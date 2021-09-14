package stringutil

// Map applies f to each string in in.
func Map(in []string, f func(string) string) []string {
	res := make([]string, 0, len(in))
	for _, s := range in {
		res = append(res, f(s))
	}
	return res
}

// Contains returns true if in contains element,
// false if not.
func Contains(in []string, element string) bool {
	for _, a := range in {
		if a == element {
			return true
		}
	}
	return false
}
