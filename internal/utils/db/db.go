package db

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Convert headers to a map
func ConvertHeadersToMap(headersRaw types.JSONRaw) (map[string]string, error) {
	// Try to parse as map[string]string directly
	var headersMap map[string]string
	if err := json.Unmarshal(headersRaw, &headersMap); err == nil {
		return headersMap, nil
	}

	// If that fails, try to parse as array of objects with name/value pairs
	var headerObjects []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(headersRaw, &headerObjects); err == nil {
		result := make(map[string]string)
		for _, h := range headerObjects {
			result[h.Name] = h.Value
		}
		return result, nil
	}

	// If that fails, try to parse as array of header names
	var headerNames []string
	if err := json.Unmarshal(headersRaw, &headerNames); err != nil {
		return nil, err
	}

	// Create a map with empty values
	headers := make(map[string]string)
	for _, name := range headerNames {
		headers[name] = ""
	}

	return headers, nil
}

func ReadFileFromRecord(app *pocketbase.PocketBase, key string) ([]byte, error) {
	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, fmt.Errorf("error creando filesystem: %w", err)
	}
	defer fsys.Close()

	blob, err := fsys.GetReader(key)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo archivo '%s': %w", key, err)
	}
	defer blob.Close()

	content, err := io.ReadAll(blob)
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo '%s': %w", key, err)
	}

	return content, nil
}

func StringToFile(content string, fileName string) (*filesystem.File, error) {
	fileObj, err := filesystem.NewFileFromBytes(
		[]byte(content),
		filepath.Base(fileName),
	)
	return fileObj, err
}
