package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

func GetRemoteIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}

	if ip != "" {
		parts := strings.Split(ip, ",")
		ip = strings.TrimSpace(parts[0])
	} else {
		parts := strings.Split(r.RemoteAddr, ":")
		if len(parts) > 1 {
			port := parts[len(parts)-1]
			ip = strings.TrimSpace(strings.Replace(r.RemoteAddr, fmt.Sprintf(":%s", port), "", -1))
		}
	}
	return ip
}
