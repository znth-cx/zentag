// Package mp4tag writes tags/cover into M4B/M4A atoms in place via go-mp4tag, avoiding ffmpeg's mov/mp4 muxer bug that corrupts cover art when tags are written in the same pass. M4BEngine remuxes chapters via ffmpeg first, then this package tags in a second, in-place pass.
package mp4tag

import (
	"fmt"

	mp4 "github.com/Sorrow446/go-mp4tag"
)

// write opens, writes tags, and closes path; var so tests can inject a fake.
var write = func(path string, tags *mp4.MP4Tags) error {
	m, err := mp4.Open(path)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Write(tags, nil); err != nil {
		return err
	}
	// go-mp4tag's final move is truncate+copy, not atomic; reopen to catch a truncated file now, not at play time.
	chk, err := mp4.Open(path)
	if err != nil {
		return fmt.Errorf("mp4tag: post-write read-back of %s failed (file may be truncated): %w", path, err)
	}
	return chk.Close()
}
