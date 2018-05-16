package api

import "strings"

// ToAuthenticationModes transforms array of authentication mode strings to valid AuthenticationModes type.
func ToAuthenticationModes(modes []string) AuthenticationModes {
	result := AuthenticationModes{}
	modesMap := map[string]bool{}

	for _, mode := range []AuthenticationMode{Token, Basic} {
		modesMap[mode.String()] = true
	}

	for _, mode := range modes {
		if _, exists := modesMap[mode]; exists {
			result.Add(AuthenticationMode(mode))
		}
	}

	return result
}

// ShouldRejectRequest returns true if url contains name and namespace of resource that should be filtered out from
// dashboard.
func ShouldRejectRequest(url string) bool {
	// For now we have only one resource that should be checked
	return strings.Contains(url, EncryptionKeyHolderName) && strings.Contains(url, EncryptionKeyHolderNamespace)
}
