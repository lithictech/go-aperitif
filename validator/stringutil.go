package validator

func mapString(in []string, f func(string) string) []string {
	res := make([]string, 0, len(in))
	for _, s := range in {
		res = append(res, f(s))
	}
	return res
}

func containsString(in []string, element string) bool {
	for _, a := range in {
		if a == element {
			return true
		}
	}
	return false
}
