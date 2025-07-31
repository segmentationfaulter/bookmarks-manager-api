package utils

import (
	"encoding/json"
	"net/http"
)

func DecodeRequestBody[T any](r *http.Request) (*T, error) {
	result := new(T)
	err := json.NewDecoder(r.Body).Decode(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
