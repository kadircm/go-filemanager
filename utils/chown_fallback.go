//go:build !linux

package utils

// MatchParentPermissions is a no-op on non-linux systems.
func MatchParentPermissions(path string) error {
	return nil
}
