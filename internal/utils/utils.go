package utils

// Contains checks if a slice contains a specific element.
// It uses type parameters to work with any slice type.
func Contains[T comparable](slice []T, element T) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}
