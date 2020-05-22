// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package atomicfile

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/text/transform"
)

// Writer returns a io.WriteCloser that writes data to a temporary file
// which gets renamed atomically as filename upon Commit.
func Writer(filename string, perm os.FileMode) (*AtomicWriter, error) {
	out, err := ioutil.TempFile(filepath.Dir(filename), ".*~")
	if err != nil {
		return nil, err
	}
	if st, err := os.Stat(filename); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		perm = st.Mode()
	}
	if err := os.Chmod(out.Name(), perm); err != nil {
		return nil, err
	}

	return &AtomicWriter{out, filename}, nil
}

// An AtomicWriter is a writer that atomically writes into a file once the Commit method is called.
type AtomicWriter struct {
	*os.File
	filename string
}

// Close closes the atomic writer. The temporary file is deleted.
// Commit cannot be called on a closed atomic writer.
func (a *AtomicWriter) Close() error {
	defer os.RemoveAll(a.Name())
	return a.File.Close()
}

// Commit closes the temporary file and renames it over the target file.
// Commit cannot be called on a closed atomic writer.
func (a *AtomicWriter) Commit() error {
	defer a.Close()
	return os.Rename(a.Name(), a.filename)
}

// WriteFrom copies data from a read into a destination file identified by filename.
// If the file already exists, it's replaced atomically with the new content and the
// original file permissions are preserved.
func WriteFrom(filename string, r io.Reader, perm os.FileMode) error {
	w, err := Writer(filename, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return w.Commit()
}

// WriteFile is a drop-in replacement for ioutil.WriteFile that writes the file atomically.
func WriteFile(filename string, data []byte, perm os.FileMode) error {
	return WriteFrom(filename, bytes.NewReader(data), perm)
}

// Transform reads the content of an existing file, passes it through a transformer and writes it back atomically.
func Transform(t transform.Transformer, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w, err := Writer(filename, 0)
	if err != nil {
		return err
	}
	defer w.Close()

	if _, err := io.Copy(w, transform.NewReader(f, t)); err != nil {
		return err
	}
	return w.Commit()
}
