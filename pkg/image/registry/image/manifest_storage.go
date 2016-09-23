package image

import (
	"io/ioutil"
	"os"
	"path"

	kapi "k8s.io/kubernetes/pkg/api"
)

type ManifestStorage interface {
	CreateOrUpdateManifest(ctx kapi.Context, id string, manifest string) error
	GetManifest(id string) (string, error)
	DeleteManifest(ctx kapi.Context, id string) error
}

type FileManifestStorage struct {
	Root string
}

func (f *FileManifestStorage) CreateOrUpdateManifest(ctx kapi.Context, id string, manifest string) error {
	err := ensureDirectory(f.Root)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(f.Root, id), []byte(manifest), 0750)
}

func (f *FileManifestStorage) GetManifest(id string) (string, error) {
	bytes, err := ioutil.ReadFile(path.Join(f.Root, id))
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (f *FileManifestStorage) DeleteManifest(ctx kapi.Context, id string) error {
	// TODO
	return nil
}

func ensureDirectory(dir string) error {
	return os.MkdirAll(dir, 0750)
}
