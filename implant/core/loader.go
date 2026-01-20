package core

import (
	"archive/tar"
	"compress/gzip"
	"embed"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed assets/*
var Assets embed.FS

func extractAssets(destDir string) error {
	entries, err := Assets.ReadDir("assets")
	if err != nil { return err }

	for _, entry := range entries {
		if entry.IsDir() { continue }
		if !strings.HasSuffix(entry.Name(), ".js") && !strings.HasSuffix(entry.Name(), ".gz") && !strings.HasSuffix(entry.Name(), ".tar.gz") {
			continue
		}

		f, err := Assets.Open("assets/" + entry.Name())
		if err != nil { return err }
		defer f.Close()

		if strings.HasSuffix(entry.Name(), ".tar.gz") {
			if err := untarGz(f, destDir); err != nil { return err }
		} else {
			out, err := os.Create(filepath.Join(destDir, entry.Name()))
			if err != nil { return err }
			io.Copy(out, f)
			out.Close()
		}
	}
	return nil
}

func untarGz(r io.Reader, dest string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil { return err }
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF { break }
		if err != nil { return err }

		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil { return err }
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil { return err }
			if _, err := io.Copy(f, tr); err != nil { return err }
			f.Close()
		}
	}
	return nil
}
