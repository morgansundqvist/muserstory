package ports

type FileReader interface {
	ReadFileContent(filePath string) (string, error)
}
