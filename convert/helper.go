package convert

func ConvertLanguageCodeToName(code string) string {
	switch code {
	case "en":
		return "English"
	case "ja":
		return "Japanese"
	case "zh", "zh-hans":
		return "Simplified Chinese"
	case "zh-tw", "zh-hant":
		return "Traditional Chinese"
	case "fr":
		return "French"
	case "it":
		return "Italian"
	default:
		return "English"
	}
}
