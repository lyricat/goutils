package convert

func ConvertLanguageCodeToName(code string) string {
	switch code {
	case "en":
		return "English"
	case "ja":
		return "Japanese"
	case "zh-hans":
		return "Simplified Chinese"
	case "zh-hant":
		return "Traditional Chinese"
	default:
		return "English"
	}
}
