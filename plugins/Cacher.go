package plugins

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/radding/harbor-plugins/proto"
	"github.com/rs/zerolog/log"
)

// CacheProvider provides the ability to cache logs and artifacts for harbor
type CacheProvider interface {
	// CreateCacheKey Provides the ability to calculate the cache key
	CreateCacheKey(context.Context, string, []string, []string) (string, error)
	Cache(context.Context, string, string, chan CacheItem) error
	ReplayCache(context.Context, string, string) (chan CacheItem, bool, error)
}

func (p *pluginClient) GetCacheKey(path string, dependencyKeys []string, additionalData []string) (string, error) {
	req := proto.CacheKeyRequest{
		LocalDirectory:     path,
		DependantCacheKeys: dependencyKeys,
		AdditionalData:     additionalData,
	}
	resp, err := p.cacheClient.CreateCacheKey(context.Background(), &req)
	if err != nil {
		return "", err
	}
	return resp.CacheKey, nil
}

func (p *pluginClient) Cache(cacheKey string, LocalCacheDirectory string, itemsToCache chan CacheItem) error {

	srv, err := p.cacheClient.Cache(context.Background())
	if err != nil {
		return errors.Wrap(err, "can't get cache server")
	}
	for item := range itemsToCache {
		req := proto.CacheRequest{
			CacheKey:            cacheKey,
			LocalCacheDirectory: LocalCacheDirectory,
			LogLine:             item.LogItem,
			ArtifactToStore:     item.ArtifactPath,
		}
		err := srv.Send(&req)
		if err != nil {
			return errors.Wrapf(err, "can't cache item {cacheKey = %s, Artifact = %s, LogLine = %s}", cacheKey, item.ArtifactPath, item.LogItem)
		}
	}
	return srv.CloseSend()
}

func (p *pluginClient) ReplayCache(cacheKey string, localCache string) (chan CacheItem, bool, error) {
	ch := make(chan CacheItem, 10)
	req := proto.ReplayRequest{
		CacheKey:            cacheKey,
		LocalCacheDirectory: localCache,
	}
	srv, err := p.cacheClient.ReplayCache(context.Background(), &req)
	if err != nil {
		close(ch)
		return ch, false, errors.Wrap(err, "failed to get first cache message")
	}
	first, err := srv.Recv()
	if err != nil {
		close(ch)
		return ch, false, errors.Wrap(err, "failed to get first cache message")
	}
	if !first.Hit {
		log.Debug().Msgf("Key not found in cache")
		close(ch)
		return ch, false, nil
	}
	go func() {
		ch <- CacheItem{
			LogItem:      first.Logs,
			ArtifactPath: first.ArtifactLocations[0],
		}
		fmt.Println("got first item to replay")
		defer close(ch)
		for {
			select {
			case <-srv.Context().Done():
				return
			default:
				msg, err := srv.Recv()
				if errors.Is(io.EOF, err) {
					return
				} else if err != nil {
					panic(err)
				}
				if msg.Err != "" {
					panic(msg.Err)
				}
				ch <- CacheItem{
					LogItem:      msg.GetLogs(),
					ArtifactPath: msg.GetArtifactLocations()[0],
				}
			}
		}

	}()
	return ch, true, err

}

func (p *pluginProvider) CreateCacheKey(ctx context.Context, cacheRequest *proto.CacheKeyRequest) (*proto.CacheKeyResponse, error) {
	if p.cachProvider == nil {
		return nil, newNotSupportedError(p.name, "Cache Provider")

	}
	newCtx := p.wrapContext(ctx, "INTERNAL:CACHER")
	req, err := p.cachProvider.CreateCacheKey(newCtx, cacheRequest.LocalDirectory, cacheRequest.DependantCacheKeys, cacheRequest.AdditionalData)
	if err != nil {
		return nil, err
	}
	return &proto.CacheKeyResponse{
		CacheKey: req,
	}, nil
}

func (p *pluginProvider) Cache(cacheSrv proto.Cacher_CacheServer) error {
	if p.cachProvider == nil {
		return newNotSupportedError(p.name, "Cache Provider")
	}
	cacheChan := make(chan CacheItem, 10)
	errChan := make(chan error)
	firstReq, err := cacheSrv.Recv()
	if err != nil {
		return errors.Wrap(err, "error getting first request")
	}
	go func() {
		newCtx := p.wrapContext(cacheSrv.Context(), "INTERNAL:CACHER")
		errChan <- p.cachProvider.Cache(newCtx, firstReq.CacheKey, firstReq.LocalCacheDirectory, cacheChan)
	}()
	cacheChan <- CacheItem{
		LogItem:      firstReq.LogLine,
		ArtifactPath: firstReq.ArtifactToStore,
	}
	for {
		select {
		case err := <-errChan:
			return errors.Wrap(err, "can't cache")
		case <-cacheSrv.Context().Done():
			return nil
		default:
			req, err := cacheSrv.Recv()
			if err != nil {
				errChan <- err
			}
			cacheChan <- CacheItem{
				LogItem:      req.LogLine,
				ArtifactPath: req.ArtifactToStore,
			}

		}
	}
}

func (p *pluginProvider) ReplayCache(req *proto.ReplayRequest, srv proto.Cacher_ReplayCacheServer) error {
	if p.cachProvider == nil {
		return newNotSupportedError(p.name, "Cache Provider")
	}
	replayChan, hit, err := p.cachProvider.ReplayCache(p.wrapContext(srv.Context(), "INTERNAL:CACHER"), req.CacheKey, req.LocalCacheDirectory)
	if err != nil {
		return errors.Wrap(err, "replay cache failed")
	}
	if !hit {
		srv.Send(&proto.ReplayResponse{
			Hit: false,
		})
		return nil
	}

	for replay := range replayChan {
		srv.Send(&proto.ReplayResponse{
			Logs:              replay.LogItem,
			ArtifactLocations: []string{replay.ArtifactPath},
			Hit:               true,
		})
	}
	return nil
}
