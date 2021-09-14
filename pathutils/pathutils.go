package pathutils

import (
	"encoding/json"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

var ErrJsonUnmarshal = errors.New("invalid json")

// Abs is an absolute path. It's just an alias of string, to avoid casting.
type Abs = string

// Rel is a relative path. It's just an alias of string, to avoid casting.
type Rel = string

// Unknown can be an absolute or relative path.
type Unknown = string

// Absdir is an absolute directory.
type Absdir Abs

// Join treats elem as the tail of the absolute directly.
// AbsDir("/foo").Join("x", "y") => "/foo/x/y"
func (kd Absdir) Join(elem ...string) Abs {
	return filepath.Join(append([]string{string(kd)}, elem...)...)
}

// ResolvePath returns path if it is an absolute path,
// or the abspath joined with base otherwise.
// Use "" for base to use the cwd, as per filepath.Abs.
func ResolvePath(base, path Unknown) (Abs, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Abs(filepath.Join(base, path))
}

func ResolvePaths(base Unknown, paths []Unknown) ([]Abs, error) {
	result := make([]Abs, len(paths))
	for i, p := range paths {
		r, err := ResolvePath(base, p)
		if err != nil {
			return nil, err
		}
		result[i] = r
	}
	return result, nil
}

// IsPathError returns true if err is present, and it or its cause is an os.PathError.
func IsPathError(err error) bool {
	return err != nil && reflect.TypeOf(errors.Cause(err)) == reflect.TypeOf(&os.PathError{})
}

// UnmarshalJsonFile unmarshals the data at path into the pointer v.
func UnmarshalJsonFile(path Unknown, v interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(v); err != nil {
		// It helps if we can determine that an error is a json error
		// The json package doesn't have a base type so wrap it here.
		return errors.Wrap(ErrJsonUnmarshal, err.Error())
	}
	return nil
}

// MarshalJsonFile marshals v into path.
func MarshalJsonFile(path Unknown, v interface{}) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// MarshalJsonFileWithDirs creates intermediate dirs and marshals v into path.
func MarshalJsonFileWithDirs(path Unknown, v interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}
	return MarshalJsonFile(path, v)
}

// CallerDir returns the directory of the calling code.
func CallerDir() Absdir {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("runtime.Caller fialed")
	}
	return Absdir(filepath.Dir(filename))
}

// TrimExt removes the extension from p.
func TrimExt(p Unknown) Unknown {
	return strings.TrimSuffix(p, filepath.Ext(p))
}

// IsDir returns true if p exists and is a directory.
func IsDir(p string) bool {
	stat, err := os.Stat(p)
	if err != nil {
		return false
	}
	return stat.IsDir()
}
