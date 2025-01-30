package util

import (
	"encoding/json"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/go-playground/validator"
)

func ReadJSONPayload(r *http.Request, body interface{}) error {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&body); err != nil {
		return err
	}
	defer r.Body.Close()

	if err := ValidatePayload(body); err != nil {
		return err
	}
	return nil
}

func ValidatePayload(body interface{}) error {
	validate := validator.New()

	validate.RegisterValidation("minrunes", minRunes)
	validate.RegisterValidation("maxrunes", maxRunes)

	if err := validate.Struct(body); err != nil {
		return err
	}

	return nil
}

func minRunes(fl validator.FieldLevel) bool {
	param := fl.Param()
	field := fl.Field().String()

	minRunes, err := strconv.Atoi(param)
	if err != nil {
		return false
	}

	return utf8.RuneCountInString(field) >= minRunes
}

func maxRunes(fl validator.FieldLevel) bool {
	param := fl.Param()
	field := fl.Field().String()

	maxRunes, err := strconv.Atoi(param)
	if err != nil {
		return false
	}

	return utf8.RuneCountInString(field) <= maxRunes
}
