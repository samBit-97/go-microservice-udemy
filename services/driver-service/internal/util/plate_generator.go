package util

import "math/rand"

// GenerateRandomPlate generates a random 3-letter license plate
func GenerateRandomPlate() string {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	plate := ""
	for range 3 {
		plate += string(letters[rand.Intn(len(letters))])
	}

	return plate
}
