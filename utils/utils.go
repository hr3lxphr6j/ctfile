package utils

import (
	"fmt"
	"regexp"
)

func Match1(re, str string) string {
	reg, err := regexp.Compile(re)
	if err != nil {
		return ""
	}
	match := reg.FindStringSubmatch(str)
	if match == nil || len(match) < 2 {
		return ""
	}
	return match[1]
}

func GetValueFromHTML(html, key string) string {
	return Match1(fmt.Sprintf(`%s="(.*?)"`, key), html)
}
