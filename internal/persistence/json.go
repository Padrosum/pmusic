package persistence

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const PrivateFileMode os.FileMode = 0o600

type DecodeError struct {
	Path string
	Err  error
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("decode %s: %v", e.Path, e.Err)
}

func (e *DecodeError) Unwrap() error { return e.Err }

// LoadJSON decodes into a temporary value and returns found=false for a
// missing file. Callers only replace live state after this function succeeds.
func LoadJSON[T any](path string, validate func(*T) error) (value T, found bool, err error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return value, false, nil
	}
	if err != nil {
		return value, false, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&value); err != nil {
		return value, true, &DecodeError{Path: path, Err: err}
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return value, true, &DecodeError{Path: path, Err: err}
	}
	if validate != nil {
		if err := validate(&value); err != nil {
			return value, true, &DecodeError{Path: path, Err: err}
		}
	}
	return value, true, nil
}

func SaveJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	return WriteFileAtomic(path, data, PrivateFileMode)
}

func WriteFileAtomic(path string, data []byte, mode os.FileMode) error {
	return writeFileAtomic(path, data, mode, nil)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode, beforeRename func(string) error) (retErr error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer func() {
		if retErr != nil {
			if err := os.Remove(tmpName); err != nil && !errors.Is(err, os.ErrNotExist) {
				retErr = errors.Join(retErr, fmt.Errorf("remove temporary file %s: %w", tmpName, err))
			}
		}
	}()

	if err := tmp.Chmod(mode); err != nil {
		return errors.Join(fmt.Errorf("set permissions for %s: %w", tmpName, err), tmp.Close())
	}
	if _, err := io.Copy(tmp, bytes.NewReader(data)); err != nil {
		return errors.Join(fmt.Errorf("write %s: %w", tmpName, err), tmp.Close())
	}
	if err := tmp.Sync(); err != nil {
		return errors.Join(fmt.Errorf("sync %s: %w", tmpName, err), tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tmpName, err)
	}
	if beforeRename != nil {
		if err := beforeRename(tmpName); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("set permissions for %s: %w", path, err)
	}
	return nil
}
