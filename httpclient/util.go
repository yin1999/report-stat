package httpclient

import (
	"bufio"
	"encoding/json"
	"os"
)

func storeJson(v interface{}, name string) error {
	file, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	enc := json.NewEncoder(writer)
	if err = enc.Encode(v); err != nil {
		return err
	}
	return writer.Flush()
}
