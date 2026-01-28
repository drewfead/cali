package auth

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/oauth2"
)

const tokenFilePermMode = 0600

// LoadToken loads an OAuth token from the specified file path
func LoadToken(tokenPath string) (*oauth2.Token, error) {
	f, err := os.Open(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open token file: %w", err)
	}
	defer f.Close()

	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	if err != nil {
		return nil, fmt.Errorf("unable to decode token: %w", err)
	}

	return tok, nil
}

// SaveToken saves an OAuth token to the specified file path with restricted permissions
func SaveToken(tokenPath string, token *oauth2.Token) error {
	f, err := os.OpenFile(tokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, tokenFilePermMode)
	if err != nil {
		return fmt.Errorf("unable to create token file: %w", err)
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(token)
	if err != nil {
		return fmt.Errorf("unable to encode token: %w", err)
	}

	return nil
}
