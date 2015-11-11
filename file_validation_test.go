package arg

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "go-arg-test")
	if err != nil {
		t.Errorf("TempDir error")
		return
	}
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "test")
	os.Create(path)

	if !FileExists(path) {
		t.Errorf("Did not indicate that existing file %s exists", path)
	}

	os.Remove(path)

	if FileExists(path) {
		t.Errorf("Did indicate that non existing file %s exists", path)
	}

	os.Mkdir(path, 0766)

	if FileExists(path) {
		t.Errorf("Did indicate that directory %s is a file", path)
	}
}

func TestDirExists(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "go-arg-test")
	if err != nil {
		t.Errorf("TempDir error")
		return
	}
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "test")
	os.Mkdir(path, 0766)

	if !DirExists(path) {
		t.Errorf("Did not indicate that existing directory %s exists", path)
	}

	os.Remove(path)

	if DirExists(path) {
		t.Errorf("Did indicate that non existing directory %s exists", path)
	}

	os.Create(path)

	if DirExists(path) {
		t.Errorf("Did indicate that file %s is a directory", path)
	}
}

func TestFileOrDirExists(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "go-arg-test")
	if err != nil {
		t.Errorf("TempDir error")
		return
	}
	defer os.RemoveAll(tmpDir)

	fpath := filepath.Join(tmpDir, "test.txt")
	dpath := filepath.Join(tmpDir, "test")
	os.Create(fpath)
	os.Mkdir(dpath, 0766)

	if !FileOrDirExists(fpath) {
		t.Errorf("Did not indicate that existing file %s exists", fpath)
	}
	if !FileOrDirExists(dpath) {
		t.Errorf("Did not indicate that existing directory %s exists", dpath)
	}

	os.Remove(fpath)
	os.Remove(dpath)

	if FileExists(fpath) {
		t.Errorf("Did indicate that non existing file %s exists", fpath)
	}
	if FileExists(dpath) {
		t.Errorf("Did indicate that non existing directory %s exists", dpath)
	}
}

func TestDirExistsOrCreate(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "go-arg-test")
	if err != nil {
		t.Errorf("TempDir error")
		return
	}
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "test1", "test2", "test3")

	ok, err := DirExistsOrCreate(path, 0766)
	if err != nil {
		t.Errorf("Error creating dir: %s", err)
	}
	if !ok {
		t.Errorf("Directory was not created")
	}
	if !DirExists(path) {
		t.Errorf("Directory %s was not created", path)
	}

	os.Remove(path)

	if DirExists(path) {
		t.Errorf("Directory %s shoult not exist", path)
	}

	ok, err = DirExistsOrCreate(path, 0766)
	if err != nil {
		t.Errorf("Error creating dir: %s", err)
	}
	if !ok {
		t.Errorf("Directory was not created")
	}
	if !DirExists(path) {
		t.Errorf("Directory %s was not created", path)
	}
}
