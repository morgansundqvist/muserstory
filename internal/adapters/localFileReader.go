package adapters

import (
	"os"

	"github.com/morgansundqvist/muserstory/internal/ports"
)

type LocalFileReader struct {
}

// NewLocalFileReader creates a new instance of LocalFileReader
func NewLocalFileReader() ports.FileReader {
	return &LocalFileReader{}
}

func (r *LocalFileReader) ReadFileContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
