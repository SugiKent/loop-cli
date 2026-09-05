// Package ui は今やるキュー画面（mvp.md「画面構成」）を Bubble Tea の Model として描く。
package ui

import (
	"fmt"
	"time"
)

// Elapsed は t から now までの経過を m / h / d で書く。切り捨て。t が now より後なら 0m。
func Elapsed(now, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d < 0:
		return "0m"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
