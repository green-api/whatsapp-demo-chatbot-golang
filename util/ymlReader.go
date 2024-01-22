package util

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
)

func GetString(keys []string) string {
	var strings, _ = getStrings()

	for i := 0; i < len(keys); i++ {
		if stringValue, ok := strings[keys[i]].(string); ok {
			return stringValue
		} else {
			strings = strings[keys[i]].(map[interface{}]interface{})
		}
	}
	return ""
}

func getStrings() (map[interface{}]interface{}, error) {
	fileContent, err := os.ReadFile("strings.yml")
	if err != nil {
		return map[interface{}]interface{}{}, fmt.Errorf("failed to read strings.yml: %v", err)
	}

	var stringsData map[interface{}]interface{}
	err = yaml.Unmarshal(fileContent, &stringsData)
	if err != nil {
		return map[interface{}]interface{}{}, fmt.Errorf("failed to unmarshal strings.yml: %v", err)
	}

	return stringsData, nil
}
