package security

const ProviderTokenVersion = 1

func EncryptProviderToken(token string) (ciphertext string, keyID string, version int, err error) {
	if _, err = EnsureAgeKey(ageKeyPath(), true); err != nil {
		return "", "", 0, err
	}
	ciphertext, keyID, err = Encrypt([]byte(token))
	if err != nil {
		return "", "", 0, err
	}
	return ciphertext, keyID, ProviderTokenVersion, nil
}

func DecryptProviderToken(ciphertext string) (string, error) {
	plaintext, err := Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
