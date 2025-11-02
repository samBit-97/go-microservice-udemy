package websocket

import (
	"errors"
	"net/http"
)

func validateUserID(r *http.Request) (string, error) {
	userID := r.URL.Query().Get("userID")
	if userID == "" {
		return "", errors.New("userID is required")
	}

	return userID, nil
}

func validateDriverParams(r *http.Request) (userID, packageSlug string, err error) {
	userID, err = validateUserID(r)
	if err != nil {
		return "", "", err
	}

	packageSlug = r.URL.Query().Get("packageSlug")
	if packageSlug == "" {
		return "", "", errors.New("packageSlug is required")
	}

	return userID, packageSlug, nil
}
