package playstore

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
)

func (p *publish) Debugf(format string, v ...any) {
	if p.verbose {
		log.Printf(format, v...)
	}
}

func (p *publish) fileExits(file string) bool {
	if _, err := p.fs.Stat(file); err == nil {
		return true
	}
	return false
}

func (p *publish) fileSize(file string) int64 {
	s, err := p.fs.Stat(file)
	if err != nil {
		log.Fatal(err)
	}
	return s.Size()
}

// TODO: must be better way to do this
func fileSha256(r io.ReadSeeker) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
