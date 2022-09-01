package main

import (
	"encoding/json"
	"io"
	"os"
)

// return ret
func GetActiveAlert(lang string) ([]map[string]interface{}, error) {
	alertsFile, err := os.Open("alerts.json")
	if err != nil {
		return nil, err
	}
	defer alertsFile.Close()
	byteValue, err := io.ReadAll(alertsFile)
	if err != nil {
		return nil, err
	}
	var alerts []map[string]interface{}
	err = json.Unmarshal(byteValue, &alerts)
	if err != nil {
		return nil, err
	}

	ret := []map[string]interface{}{}
	for _, a := range alerts {
		// convert to bool
		active := a["active"].(bool)
		if active {
			if lang == "id" {
				if _, ok := a["iDD"]; ok {
					lang = "iDD"
				}
			} else if _, ok := a[lang]; !ok {
				lang = "en"
			}
			retItem := map[string]interface{}{
				"id":       a["id"],
				"priority": a["priority"],
				"active":   a["active"],
			}
			if _, ok := a["timestamp"]; ok {
				retItem["timestamp"] = a["timestamp"]
			}
			if _, ok := a["link"]; ok {
				retItem["link"] = a["link"]
			}
			for k, v := range a[lang].(map[string]interface{}) {
				retItem[k] = v
			}
			ret = append(ret, retItem)
		}
	}
	return ret, nil
}
