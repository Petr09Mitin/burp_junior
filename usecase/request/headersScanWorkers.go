package request

import (
	"context"
	"maps"
	"sync"

	"github.com/burp_junior/domain"
)

func (r *RequestService) scanHeadersWorker(ctx context.Context, wg *sync.WaitGroup, req *domain.SafeHTTPRequest, ci SafeInjections, headers *sync.Map) {
	defer wg.Done()

	headersWg := &sync.WaitGroup{}
	headersWg.Add(len(req.Headers.Sam))

	req.Headers.Mu.RLock()
	for key := range req.Headers.Sam {
		go r.scanHeaderWorker(ctx, headersWg, req, key, ci, headers)
	}

	req.Headers.Mu.RUnlock()

	headersWg.Wait()
}

func (r *RequestService) scanHeaderWorker(ctx context.Context, wg *sync.WaitGroup, safeReq *domain.SafeHTTPRequest, key string, ci SafeInjections, headers *sync.Map) {
	defer wg.Done()
	headersScansWg := &sync.WaitGroup{}
	ci.mu.RLock()
	headersScansWg.Add(len(ci.ci))

	for _, scan := range ci.ci {
		go r.sendHeaderScan(ctx, headersScansWg, *safeReq, key, scan, headers)
	}
	ci.mu.RUnlock()

	headersScansWg.Wait()
}

func (r *RequestService) sendHeaderScan(ctx context.Context, wg *sync.WaitGroup, safeReq domain.SafeHTTPRequest, key string, scan string, headers *sync.Map) {
	defer wg.Done()
	dirty := domain.SafeStringArrMap{
		Sam: map[string][]string{},
		Mu:  &sync.RWMutex{},
	}
	safeReq.Headers.Mu.RLock()
	maps.Copy(dirty.Sam, safeReq.Headers.Sam)
	safeReq.Headers.Mu.RUnlock()
	dirty.Sam[key] = []string{scan}
	safeReq.Headers = dirty
	res, err := r.SendHTTPRequest(ctx, domain.MakeHTTPRequestFromSafe(&safeReq))
	if err != nil {
		return
	}

	if r.isCommandInjectionVulnerable(res) {
		val, _ := headers.LoadOrStore(key, []string{})
		headers.Store(key, append(val.([]string), scan))
	}
}
