package runners

import (
	"bufio"
	"io"
	"os"
	"sort"

	"github.com/pkg/errors"
	plugins "github.com/radding/harbor-plugins"
	"github.com/rs/zerolog/log"
)

// Cacher is the interface for caching task runs and builds
type Cacher interface {
	// CalculateCacheKey takes the RunRecipe and some optional additional data, and then calculates the cache key
	// based on the run recipe to get a unique id. If one of the elements changes, then the whole cache key changes,
	// triggering a new run
	CalculateCacheKey(r *RunRecipe, additionalData ...string) (string, error)
	// ReplayCachedLogs looks in our cache for entries with cache key and attempts to write those log contents into
	// w. Returns true if the logs were found in the cache, false otherwise.
	ReplayCachedLogs(cacheKey string, w io.Writer) (bool, error)
	// WriteLogsToCache takes a cacheKey and reads data from r to write the logs to a file in our cache directory
	WriteLogsToCache(cacheKey string, r io.Reader) error
}

type cacher struct {
	packageToHashKey map[string]string
	cacherClient     plugins.PluginClient
	localCacheDir    string
	// dirToHashKey     map[string]string
}

func newCacher(plugin plugins.PluginClient, localCacheDir string) Cacher {
	return &cacher{
		packageToHashKey: map[string]string{},
		cacherClient:     plugin,
		localCacheDir:    localCacheDir,
		// dirToHashKey:     map[string]string{},
	}
}

func openFile(fileName string) (io.ReadCloser, error) {
	return os.Open(fileName)
}

func (c *cacher) CalculateCacheKey(r *RunRecipe, additionalData ...string) (string, error) {
	return c.calculateCacheKey(r, additionalData, openFile)
}

func (c *cacher) calculateCacheKey(r *RunRecipe, additionalData []string, fileOpener func(string) (io.ReadCloser, error)) (string, error) {
	if c.cacherClient == nil {
		return "", nil
	}
	hashKey, ok := c.packageToHashKey[r.pkgObject.Name]
	if ok {
		return hashKey, nil
	}
	deps := r.Needs[:]
	// Sort the deps so its stable every time
	sort.Slice(deps, func(i, j int) bool {
		depI := r.Needs[i]
		depJ := r.Needs[j]
		return depI.HashKey() > depJ.HashKey()
	})
	childKeys := []string{}
	for _, dep := range deps {
		childKey, err := c.calculateCacheKey(dep, additionalData, fileOpener)
		if err != nil {
			return "", errors.Wrap(err, "can't calculate cache key")
		}
		childKeys = append(childKeys, childKey)
	}

	hashKey, err := c.cacherClient.GetCacheKey(r.pkgObject.WorkspaceRoot(), childKeys, additionalData)
	if err != nil {
		return "", errors.Wrap(err, "can't get cache key")
	}
	return hashKey, nil
}

func (c *cacher) ReplayCachedLogs(cacheKey string, w io.Writer) (bool, error) {
	if c.cacherClient == nil {
		return false, nil
	}
	log.Debug().Msg("Waiting to replay")
	ch, hit, err := c.cacherClient.ReplayCache(cacheKey, c.localCacheDir)
	if err != nil {
		return false, errors.Wrap(err, "can't replay cache")
	} else if !hit {
		log.Debug().Msg("No cahce key found")
		return hit, nil
	}
	log.Debug().Msg("Replay kicked off")
	for item := range ch {
		if item.LogItem != "" {
			w.Write([]byte(item.LogItem))
		}
	}
	return hit, nil
}

func (c *cacher) WriteLogsToCache(cacheKey string, r io.Reader) error {
	if c.cacherClient == nil {
		return nil
	}
	ch := make(chan plugins.CacheItem)
	defer close(ch)
	done := make(chan struct{})
	errCh := make(chan error)

	go func() {
		defer close(done)
		defer close(errCh)
		err := c.cacherClient.Cache(cacheKey, c.localCacheDir, ch)
		if err != nil {
			errCh <- errors.Wrap(err, "can't cache")
			return
		}
		done <- struct{}{}
	}()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		select {
		case <-done:
			return nil
		case err := <-errCh:
			return err
		default:
			logLine := scanner.Text()
			ch <- plugins.CacheItem{
				LogItem: logLine,
			}
		}
	}
	return nil
}
