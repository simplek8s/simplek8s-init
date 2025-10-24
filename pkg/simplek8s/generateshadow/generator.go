// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package generateshadow creates a hashed password for use in Linux systems.
package generateshadow

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"strings"

	"simplek8s/pkg/simplek8s"

	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
)

// consonants and vowels used to create pronounceable pseudo-words.
//
// Keeping tables small saves space while still producing readable output.
var consonants = []string{
	"b", "c", "d", "f", "g", "h", "j", "k", "l", "m", "n", "p", "r", "s", "t", "v", "w", "y", "z",
}
var vowels = []string{"a", "e", "i", "o", "u", "ea", "ai", "oo", "ou"}

func randChoose(list []string) (string, error) {
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(list))))
	if err != nil {
		return "", err
	}
	return list[idx.Int64()], nil
}

// generateWord creates a pseudo-English word formed by syllables.
//
// Each syllable is roughly consonant + vowel + optional consonant.
func generateSillyWord(syllables int) (string, error) {
	var b strings.Builder
	for i := 0; i < syllables; i++ {
		c, err := randChoose(consonants)
		if err != nil {
			return "", err
		}
		v, err := randChoose(vowels)
		if err != nil {
			return "", err
		}
		b.WriteString(c)
		b.WriteString(v)
		// Optional closing consonant every second syllable
		if i%2 == 0 {
			if c2, err := randChoose(consonants); err == nil {
				b.WriteString(c2)
			}
		}
	}
	return b.String(), nil
}

// GeneratePassphrase builds a passphrase composed of several pseudo-words.
//
// Each word is built from multiple syllables to increase entropy.
// Example: "vakeel-ponadi-lurek-samofi"
func GeneratePronounceablePassphrase(words, sylPerWord int) ([]string, error) {
	out := make([]string, words)
	for i := range words {
		w, err := generateSillyWord(sylPerWord)
		if err != nil {
			return nil, err
		}
		out[i] = w
	}
	return out, nil
}

// generateSalt creates a random salt up to 16 characters using the same alphabet as /etc/shadow
func GenerateSalt() (string, error) {
	const saltChars = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	buf := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = saltChars[int(buf[i])%len(saltChars)]
	}
	return string(buf), nil
}

func GenerateShadowPassword(password string) (string, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return "", fmt.Errorf("cannot generate salt: %w", err)
	}

	salt = fmt.Sprintf("$6$%s$", salt)

	crypter := sha512_crypt.New()
	hash, err := crypter.Generate([]byte(password), []byte(salt))
	if err != nil {
		return "", fmt.Errorf("cannot generate hash: %w", err)
	}

	return hash, nil
}

// Generates a password.
// defaultPwd will be used when debug is true.
func GeneratePwd(defaultPwd string) (plain string, hashed string, err error) {
	if !simplek8s.IsDebug() {
		// generate a secure password.
		words, err := GeneratePronounceablePassphrase(4, 2)
		if err != nil {
			return "", "", fmt.Errorf("cannot generate passphrase: %w", err)
		}
		defaultPwd = strings.Join(words, "-")
	}

	hash, err := GenerateShadowPassword(defaultPwd)
	if err != nil {
		return "", "", fmt.Errorf("cannot generate password: %w", err)
	}

	return defaultPwd, hash, nil
}
