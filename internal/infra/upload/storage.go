package upload

import "mime/multipart"

type Storage struct {
	basePath string
}

func (s *Storage) ReceiveFile(up *multipart.FileHeader) error {
	panic("implement me!")
}
