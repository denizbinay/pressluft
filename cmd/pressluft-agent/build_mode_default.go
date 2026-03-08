//go:build !dev

package main

func isDevBuild() bool {
	return false
}
