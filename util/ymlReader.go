package util

import (
	"gopkg.in/yaml.v2"
	"os"
)

func GetString(keys []string) string {
	var strings, _ = getStrings()

	for i := 0; i < len(keys); i++ {
		if stringValue, ok := strings[keys[i]].(string); ok {
			return stringValue
		} else {
			if stringsRaw, ok := strings[keys[i]].(map[interface{}]interface{}); ok {
				strings = stringsRaw
			} else {
				panic("failed to read strings.yaml: " + keys[i] + " is nil")
			}
		}
	}
	return ""
}

func getStrings() (map[interface{}]interface{}, error) {
	fileContent, err := os.ReadFile("strings.yaml")
	if err != nil {
		panic("Not found strings.yaml: " + err.Error())
	}

	var stringsData map[interface{}]interface{}
	err = yaml.Unmarshal(fileContent, &stringsData)
	if err != nil {
		panic("failed to unmarshal strings.yaml: " + err.Error())
	}

	return stringsData, nil
}
