package files

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UploadMeta is the JSON header written before the tar.gz payload.
type UploadMeta struct {
	RepoName     string `json:"repo_name"`
	Branch       string `json:"branch"`
	InstanceID   string `json:"instance_id"`
	InstanceName string `json:"instance_name"`
	PayloadSize  int64  `json:"payload_size"`
}

// WriteHeader encodes meta as a JSON line to w.
func WriteHeader(w io.Writer, meta UploadMeta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

// PullMeta is the JSON header the server sends before the pull tar.gz payload.
type PullMeta struct {
	Type       string `json:"type"` // "ok" or "error"
	InstanceID string `json:"instance_id"`
	RepoName   string `json:"repo_name"`
	Message    string `json:"message,omitempty"`
}

// WritePullHeader encodes meta as a JSON line to w.
func WritePullHeader(w io.Writer, meta PullMeta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

// ReadPullHeader reads the first JSON line from r as PullMeta.
func ReadPullHeader(r *bufio.Reader) (*PullMeta, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read pull header: %w", err)
	}
	var meta PullMeta
	if err := json.Unmarshal(line, &meta); err != nil {
		return nil, fmt.Errorf("parse pull header: %w", err)
	}
	return &meta, nil
}

// ReadHeader reads the first JSON line from r as UploadMeta.
func ReadHeader(r *bufio.Reader) (*UploadMeta, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	var meta UploadMeta
	if err := json.Unmarshal(line, &meta); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}
	return &meta, nil
}

// Pack creates a tar.gz of selectedFiles (relative to root) and writes it to w.
func Pack(root string, selectedFiles []string, w io.Writer) error {
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)

	for _, relPath := range selectedFiles {
		fullPath := filepath.Join(root, relPath)
		info, err := os.Lstat(fullPath)
		if err != nil {
			return fmt.Errorf("stat %s: %w", relPath, err)
		}
		if info.IsDir() {
			continue
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("tar header %s: %w", relPath, err)
		}
		hdr.Name = relPath // use relative path inside archive

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		f, err := os.Open(fullPath)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, f)
		f.Close()
		if copyErr != nil {
			return copyErr
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

// Unpack extracts a tar.gz from r into destDir, preventing path traversal.
func Unpack(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}

		// Security: reject absolute paths and traversal
		clean := filepath.Clean(hdr.Name)
		if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
			return fmt.Errorf("unsafe path in archive: %s", hdr.Name)
		}

		target := filepath.Join(destDir, clean)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)&0755|0111); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0755|0644)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(f, tr)
			f.Close()
			if copyErr != nil {
				return copyErr
			}
		}
	}
	return nil
}
