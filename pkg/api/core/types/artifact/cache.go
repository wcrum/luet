// Copyright © 2021 Ettore Di Giacinto <mudler@mocaccino.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.

package artifact

import (
	"crypto/sha512"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// OutputID represents a cache entry identifier (SHA512 hash)
type OutputID [64]byte

// Cache represents a simple file cache implementation
type Cache struct {
	dir string
}

// ArtifactCache wraps the Cache for artifact-specific operations
type ArtifactCache struct {
	Cache
}

func NewCache(dir string) *ArtifactCache {
	return &ArtifactCache{Cache: Cache{dir: dir}}
}

func (c *ArtifactCache) cacheID(a *PackageArtifact) [64]byte {
	fingerprint := filepath.Base(a.Path)
	if a.CompileSpec != nil && a.CompileSpec.Package != nil {
		fingerprint = a.CompileSpec.Package.GetFingerPrint()
	}
	if len(a.Checksums) > 0 {
		for _, cs := range a.Checksums.List() {
			t := cs[0]
			result := cs[1]
			fingerprint += fmt.Sprintf("+%s:%s", t, result)
		}
	}
	return sha512.Sum512([]byte(fingerprint))
}

// GetFile retrieves a file from the cache by its ID
func (c *Cache) GetFile(id [64]byte) (string, bool, error) {
	// Convert the hash to a hex string for the filename
	filename := fmt.Sprintf("%x", id)
	filepath := filepath.Join(c.dir, filename)

	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return "", false, errors.New("file not found in cache")
	}

	return filepath, true, nil
}

// Put stores a file in the cache
func (c *Cache) Put(id [64]byte, reader io.Reader) (OutputID, int64, error) {
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return OutputID{}, 0, errors.Wrapf(err, "failed to create cache directory %s", c.dir)
	}

	filename := fmt.Sprintf("%x", id)
	filepath := filepath.Join(c.dir, filename)

	outFile, err := os.Create(filepath)
	if err != nil {
		return OutputID{}, 0, errors.Wrapf(err, "failed to create cache file %s", filepath)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, reader)
	if err != nil {
		os.Remove(filepath)
		return OutputID{}, 0, errors.Wrapf(err, "failed to copy content to cache file %s", filepath)
	}

	return OutputID(id), written, nil
}

func (c *ArtifactCache) Get(a *PackageArtifact) (string, error) {
	fileName, _, err := c.Cache.GetFile(c.cacheID(a))
	return fileName, err
}

func (c *ArtifactCache) Put(a *PackageArtifact) (OutputID, int64, error) {
	file, err := os.Open(a.Path)
	if err != nil {
		return OutputID{}, 0, errors.Wrapf(err, "failed opening %s", a.Path)
	}
	defer file.Close()
	return c.Cache.Put(c.cacheID(a), file)
}
