package finalizer

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}

	return false
}

func RemoveString(slice []string, s string) []string {
	var result []string

	for _, item := range slice {
		if item == s {
			continue
		}

		result = append(result, item)
	}

	return result
}
