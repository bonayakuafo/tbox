package core

import (
	"encoding/json"
	"os"
)

// WriteJSON writes the object as JSON to the given path.
func WriteJSON(v interface{}, path string) error {
	file, e := os.Create(path)
	if e != nil {
		return e
	}
	defer file.Close()
	jsonEncode := json.NewEncoder(file)
	jsonEncode.SetIndent("", "\t")
	return jsonEncode.Encode(v)
}
