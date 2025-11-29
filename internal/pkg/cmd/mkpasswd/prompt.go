package mkpasswd

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

func PromptSecret(label string) (string, error) {
	fmt.Print(label)
	s, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Println()
	return string(s), nil
}
