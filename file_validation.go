package arg

import (
	"os"
)

// FileExists returns true if a file exists at path, false otherwise.
func FileExists(path string) bool {
	exists, f := exists(path)
	if exists {
		if !f.IsDir() {
			return true
		}
	}
	return false
}

// DirExists returns true if a directory exists at path, false otherwise.
func DirExists(path string) bool {
	exists, d := exists(path)
	if exists {
		if d.IsDir() {
			return true
		}
	}
	return false
}

// FileOrDirExists returns true if either a file or directory exists at path,
// false otherwise.
func FileOrDirExists(path string) bool {
	exists, _ := exists(path)
	return exists
}

// DirExistsOrCreate checks if a directory exists or path, if not it attempts
// to create it, and all parent directories, using os.MkdirAll.
// Returns ok if the file exists or if it could be created and error
// if something goes wrong when creating the directory.
func DirExistsOrCreate(path string, perm os.FileMode) (bool, error) {
	if DirExists(path) {
		return true, nil
	}

	err := os.MkdirAll(path, perm)
	if err != nil {
		return DirExists(path), err
	}

	return DirExists(path), nil
}

// exists returns true if a file or directory exists at path, false otherwise.
func exists(path string) (bool, os.FileInfo) {
	f, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
	}
	return true, f
}
