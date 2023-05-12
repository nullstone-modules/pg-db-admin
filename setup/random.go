package setup

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sort"
)

func randomRoleCreds(usernamePrefix string) (string, string, error) {
	usernameSuffix, err := randomString(5, "")
	if err != nil {
		return "", "", fmt.Errorf("error generating username: %w", err)
	}
	passwordRaw, err := randomString(16, "!#$%&*()-_=+[]{}<>:?")
	if err != nil {
		return "", "", fmt.Errorf("error generating password: %w", err)
	}
	return fmt.Sprintf("%s_%s", usernamePrefix, string(usernameSuffix)), string(passwordRaw), nil
}

func randomString(length int64, specialChars string) ([]byte, error) {
	const numChars = "0123456789"
	const lowerChars = "abcdefghijklmnopqrstuvwxyz"
	const upperChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var result []byte

	var chars = upperChars + lowerChars + numChars + specialChars

	result = make([]byte, 0, length)

	s, err := generateRandomBytes(&chars, length)
	if err != nil {
		return nil, err
	}

	result = append(result, s...)

	order := make([]byte, len(result))
	if _, err := rand.Read(order); err != nil {
		return nil, err
	}

	sort.Slice(result, func(i, j int) bool {
		return order[i] < order[j]
	})

	return result, nil
}

func generateRandomBytes(charSet *string, length int64) ([]byte, error) {
	bytes := make([]byte, length)
	setLen := big.NewInt(int64(len(*charSet)))
	for i := range bytes {
		idx, err := rand.Int(rand.Reader, setLen)
		if err != nil {
			return nil, err
		}
		bytes[i] = (*charSet)[idx.Int64()]
	}
	return bytes, nil
}
