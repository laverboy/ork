package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ZipFiles(filename string, files []string) error {
	newZipFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	// Add files to zip
	for _, file := range files {
		if err = AddFileToZip(zipWriter, file); err != nil {
			return err
		}
	}
	return nil
}

func AddFileToZip(zw *zip.Writer, src string) error {
	originalParent := ""
	if strings.Contains(src, "resources") {
		originalParent = "resources/"
	}
	if strings.Contains(src, "cosmos") {
		originalParent = "cosmos/"
	}

	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		header, err := zip.FileInfoHeader(fi)
		if err != nil {
			return err
		}

		if isSymlink(fi) {
			// file body for symlinks is the symlink target
			linkTarget, err := os.Readlink(file)
			if err != nil {
				return fmt.Errorf("%s: readlink: %v", fi.Name(), err)
			}

			linkPath := filepath.Join(filepath.Dir(file), linkTarget)

			fileInfo, err := os.Stat(linkPath)
			if err != nil {
				return err
			}

			h, err := zip.FileInfoHeader(fileInfo)
			if err != nil {
				return err
			}

			header = h
			file = linkPath
		}

		// need to add parent dir for required dirs back in
		header.Name = originalParent + header.Name

		header.Method = zip.Store
		// write the header
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		// open files for zipping
		f, err := os.Open(file)
		if err != nil {
			return err
		}

		// copy file data into zip writer
		if _, err := io.Copy(writer, f); err != nil {
			return err
		}

		// manually close here after each file operation; deferring would cause each file close
		// to wait until all operations have completed.
		f.Close()

		return nil
	})
}

func isSymlink(fi os.FileInfo) bool {
	return fi.Mode()&os.ModeSymlink != 0
}
