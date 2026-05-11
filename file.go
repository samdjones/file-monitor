package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func handleFile(path string, cfg Config) error {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if info.IsDir() {
		return nil
	}
	if !matchesFilter(path, cfg.Exts) {
		return nil
	}

	dstName := buildDstName(filepath.Base(path), cfg)
	dstPath := filepath.Join(cfg.Dst, dstName)

	if filesIdentical(path, dstPath) {
		log.Printf("Skipped %q (identical to destination)", filepath.Base(path))
		return nil
	}

	if err := copyFile(path, dstPath); err != nil {
		return fmt.Errorf("copy %q to %q: %w", path, dstPath, err)
	}
	log.Printf("Copied  %q to %q", filepath.Base(path), dstPath)

	if cfg.Delete {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("delete %q: %w", path, err)
		}
		log.Printf("Deleted %q", path)
	}
	return nil
}

func buildDstName(name string, cfg Config) string {
	if !cfg.Rename {
		return name
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	ts := time.Now().Format(cfg.Pattern)
	return base + "_" + ts + ext
}

func matchesFilter(path string, exts []string) bool {
	if len(exts) == 0 {
		return true
	}
	fileExt := strings.ToLower(filepath.Ext(path))
	for _, e := range exts {
		if fileExt == e {
			return true
		}
	}
	return false
}

func hashFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func filesIdentical(src, dst string) bool {
	dstHash, err := hashFile(dst)
	if err != nil {
		return false
	}
	srcHash, err := hashFile(src)
	if err != nil {
		return false
	}
	if len(srcHash) != len(dstHash) {
		return false
	}
	for i := range srcHash {
		if srcHash[i] != dstHash[i] {
			return false
		}
	}
	return true
}

func copyFile(src, dst string) (retErr error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); retErr == nil {
			retErr = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// handleFileWithRetry handles file copying with retry logic for removable destination volumes.
func handleFileWithRetry(path string, cfg Config) error {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if info.IsDir() {
		return nil
	}
	if !matchesFilter(path, cfg.Exts) {
		return nil
	}

	dstName := buildDstName(filepath.Base(path), cfg)

	for {
		dstPath := filepath.Join(cfg.Dst, dstName)

		if filesIdentical(path, dstPath) {
			log.Printf("Skipped %q (identical to destination)", filepath.Base(path))
			return nil
		}

		if err := copyFile(path, dstPath); err == nil {
			log.Printf("Copied  %q → %q", filepath.Base(path), dstPath)

			if cfg.Delete {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("delete %q: %w", path, err)
				}
				log.Printf("Deleted %q", path)
			}
			return nil
		} else if cfg.DestVolumeName != "" && isDestinationError(err) {
			// Destination volume missing, wait for it to remount
			log.Printf("Destination unavailable: %v, waiting for volume %q to be mounted...", err, cfg.DestVolumeName)
			for {
				newDst := resolveDest(cfg.DestVolumeName, cfg.DestVolumePath)
				cfg.Dst = newDst
				if _, err := os.Stat(newDst); err == nil {
					log.Printf("Destination volume remounted, retrying copy...")
					break
				}
				time.Sleep(2 * time.Second)
			}
		} else {
			return fmt.Errorf("copy %q to %q: %w", path, cfg.Dst, err)
		}
	}
}

// isDestinationError checks if an error is related to destination unavailability.
func isDestinationError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common drive/path not found errors
	errStr := err.Error()
	return strings.Contains(errStr, "no such file") || strings.Contains(errStr, "The system cannot find the path") || strings.Contains(errStr, "permission denied")
}
