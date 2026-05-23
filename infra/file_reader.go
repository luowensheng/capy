package infra

import "os"

type FileReader struct{}

func (FileReader) Read(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (FileReader) Write(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
