package checksum

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"

	log "github.com/sirupsen/logrus"
)

func Sha256sum(r io.Reader) (string, error) {
	log.WithField("func", "Sha256sum").Debug("start")
	defer log.WithField("func", "Sha256sum").Debug("end")

	if r == nil {
		err := errors.New("`r` is nil")
		log.Error(err)
		return "", err
	}

	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		log.Error(err)
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
