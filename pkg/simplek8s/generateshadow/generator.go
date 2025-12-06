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
	rnd "math/rand"
	"strings"
	"unicode"

	"github.com/simplek8s/simplek8s-init/pkg/simplek8s"

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

// RandomUppercase randomizes uppercase conversion of n characters within a string.
func RandomUppercase(s string, n int) string {
	runes := []rune(s)
	length := len(runes)

	if n <= 0 || length == 0 {
		return s
	}

	if n > length {
		n = length
	}

	// Create a list of all positions
	indices := rnd.Perm(length)[:n]

	for _, idx := range indices {
		runes[idx] = unicode.ToUpper(runes[idx])
	}

	return string(runes)
}

// LeetMap holds character substitutions.
var LeetMap = map[rune]rune{
	'A': '4', 'a': '4',
	'E': '3', 'e': '3',
	'I': '1', 'i': '1',
	'O': '0', 'o': '0',
	'S': '5', 's': '5',
	'T': '7', 't': '7',
	'B': '8', 'b': '8',
	'G': '6', 'g': '6',
	'Z': '2', 'z': '2',
}

// ReplacementList stores all allowed substitutions for a rune.
var ReplacementList = map[rune][]rune{
	'A': {'4', '@'}, 'a': {'4', '@'},
	'E': {'3', '&'}, 'e': {'3', '&'},
	'I': {'1', '!'}, 'i': {'1', '!'},
	'O': {'0', '*'}, 'o': {'0', '*'},
	'S': {'5', '$'}, 's': {'5', '$'},
	'T': {'7', '+'}, 't': {'7', '+'},
	'B': {'8', 'ß'}, 'b': {'8', 'ß'},
	'G': {'6', '9'}, 'g': {'6', '9'},
	'Z': {'2', '%'}, 'z': {'2', '%'},
	'H': {'#'}, 'h': {'#'},
	'C': {'('}, 'c': {'('},
	'K': {'<'}, 'k': {'<'},
	'X': {'%'}, 'x': {'%'},
	'Q': {'?'}, 'q': {'?'},
}

// RandomDecorate randomly transforms n characters using ReplacementList.
func RandomDecorate(s string, n int) string {
	runes := []rune(s)
	length := len(runes)

	if n <= 0 || length == 0 {
		return s
	}

	// Find eligible indices
	indices := make([]int, 0)
	for i, r := range runes {
		if _, ok := ReplacementList[r]; ok {
			indices = append(indices, i)
		}
	}

	if len(indices) == 0 {
		return s
	}
	if n > len(indices) {
		n = len(indices)
	}

	rnd.Shuffle(len(indices), func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })

	// Replace characters
	for _, idx := range indices[:n] {
		reps := ReplacementList[runes[idx]]
		runes[idx] = reps[rnd.Intn(len(reps))]
	}

	return string(runes)
}

func GeneratePwd() (string, error) {
	nWords := 3
	// generate a secure password.
	words, err := GeneratePronounceablePassphrase(nWords, 2)
	if err != nil {
		return "", fmt.Errorf("cannot generate passphrase: %w", err)
	}
	pwd := strings.Join(words, "-")
	pwd = RandomUppercase(pwd, 1)
	pwd = RandomDecorate(pwd, 1)
	return pwd, nil
}

// Generates a password.
// defaultPwd will be used when debug is true.
func GeneratePwdWithDefault(defaultPwd string) (plain string, hashed string, err error) {
	if !simplek8s.IsDebug() {
		defaultPwd, err = GeneratePwd()
		if err != nil {
			return "", "", err
		}
	}

	hash, err := GenerateShadowPassword(defaultPwd)
	if err != nil {
		return "", "", fmt.Errorf("cannot generate password: %w", err)
	}

	return defaultPwd, hash, nil
}
