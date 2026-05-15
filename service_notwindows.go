//go:build !windows

package main

func isService() bool        { return false }
func runService(_ func()) error { return nil }
