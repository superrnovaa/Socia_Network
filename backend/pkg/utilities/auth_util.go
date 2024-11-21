package utilities

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

func SaveFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	// Create the uploads directory if it doesn't exist
	uploadsDir := "../../pkg/db/uploads"
	if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
		return "", err
	}

	// Generate a unique filename
	fileExt := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), fileExt)

	// Create a new file in the uploads directory with the unique filename
	filePath := filepath.Join(uploadsDir, filename)
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// Copy the uploaded file to the destination file
	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}

	// Return the new filename
	return filename, nil
}
