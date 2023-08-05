package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"
	plugins "github.com/radding/harbor-plugins"
	"golang.org/x/net/context"
)

type localCacher struct {
	dirToHashKey map[string]string
	fileOpener   opener
}

type opener func(fileName string) (io.ReadCloser, error)

func newCacher(f opener) plugins.CacheProvider {
	return &localCacher{
		dirToHashKey: map[string]string{},
		fileOpener:   f,
	}
}

func (c *localCacher) CreateCacheKey(ctx context.Context, dir string, dependencyKeys []string, additionalData []string) (string, error) {
	logger := ctx.Value("Logger").(hclog.Logger)
	logger.Trace(fmt.Sprintf("Calculating cache key for %s", dir))
	dirHash, ok := c.dirToHashKey[dir]
	if !ok {
		hasher := md5.New()
		files := []string{}
		err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			return "", errors.Wrap(err, "can't walk directory")
		}
		sort.Strings(files)
		for _, i := range files {
			file, err := c.fileOpener(i)
			if err != nil {
				return "", errors.Wrapf(err, "can't open file %s", i)
			}
			defer file.Close()
			io.Copy(hasher, file)
		}
		dirHash = fmt.Sprintf("%x", hasher.Sum([]byte{}))
		c.dirToHashKey[dir] = dirHash
	}
	hasher := md5.New()
	hasher.Write([]byte(dirHash))
	for _, key := range dependencyKeys {
		hasher.Write([]byte(key))
	}
	for _, key := range additionalData {
		hasher.Write([]byte(key))
	}
	// get the cache key of all children
	hashKey := fmt.Sprintf("%x", hasher.Sum([]byte{}))
	return hashKey, nil
}

func (c *localCacher) Cache(ctx context.Context, cacheKey, localCacheDir string, ch chan plugins.CacheItem) error {
	logger := ctx.Value("Logger").(hclog.Logger)
	logger.Trace("Beginning caching")
	os.MkdirAll(filepath.Join(localCacheDir, cacheKey), 0755)
	logPath := filepath.Join(localCacheDir, cacheKey, "cached.log")
	logger.Debug(fmt.Sprintf("Adding cache file at %s", logPath))
	logFi, err := os.OpenFile(logPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return errors.Wrapf(err, "umable to get log file at %s", logPath)
	}
	for item := range ch {
		if item.LogItem != "" {
			_, err := logFi.WriteString(item.LogItem)
			if err != nil {
				return errors.Wrap(err, "can't save log item")
			}
		}
		if item.ArtifactPath != "" {
			logger.Debug(fmt.Sprintf("Copying artifact to local cache"))
		}
	}
	return nil
}

func (c *localCacher) ReplayCache(ctx context.Context, cacheKey string, localCache string) (chan plugins.CacheItem, bool, error) {
	logger := ctx.Value("Logger").(hclog.Logger)
	ch := make(chan plugins.CacheItem)
	fiPath := filepath.Join(localCache, cacheKey, "cached.log")
	openFile, err := os.Open(fiPath)
	logger.Debug(fmt.Sprintf("Got Path for cache key %s: %s", cacheKey, fiPath))
	if errors.Is(err, os.ErrNotExist) {
		logger.Debug("Cache file does not exsist")
		close(ch)
		return ch, false, nil
	} else if err != nil {
		close(ch)
		logger.Debug(fmt.Sprintf("Failed to read cache: %s", err))
		return ch, false, errors.Wrap(err, "failed to open cache file")
	}
	go func() {
		defer close(ch)
		lineReader := bufio.NewScanner(openFile)
		for lineReader.Scan() {
			logLine := lineReader.Text()
			ch <- plugins.CacheItem{
				LogItem: logLine,
			}
		}
	}()
	return ch, true, nil
}
