package request

import (
	"context"
	"maps"
	"sync"

	"github.com/burp_junior/domain"
)

func (r *RequestService) scanPostParamsWorker(ctx context.Context, wg *sync.WaitGroup, req *domain.SafeHTTPRequest, ci SafeInjections, postParams *sync.Map) {
	defer wg.Done()

	postParamsWg := &sync.WaitGroup{}
	postParamsWg.Add(len(req.PostParams.Sam))

	req.PostParams.Mu.RLock()
	for key := range req.PostParams.Sam {
		go r.scanPostParamWorker(ctx, postParamsWg, req, key, ci, postParams)
	}

	req.PostParams.Mu.RUnlock()

	postParamsWg.Wait()
}

func (r *RequestService) scanPostParamWorker(ctx context.Context, wg *sync.WaitGroup, safeReq *domain.SafeHTTPRequest, key string, ci SafeInjections, postParams *sync.Map) {
	defer wg.Done()
	postParamsScansWg := &sync.WaitGroup{}
	ci.mu.RLock()
	postParamsScansWg.Add(len(ci.ci))

	for _, scan := range ci.ci {
		go r.sendPostParamScan(ctx, postParamsScansWg, *safeReq, key, scan, postParams)
	}
	ci.mu.RUnlock()

	postParamsScansWg.Wait()
}

func (r *RequestService) sendPostParamScan(ctx context.Context, wg *sync.WaitGroup, safeReq domain.SafeHTTPRequest, key string, scan string, postParams *sync.Map) {
	defer wg.Done()
	dirty := domain.SafeStringArrMap{
		Sam: map[string][]string{},
		Mu:  &sync.RWMutex{},
	}
	safeReq.PostParams.Mu.RLock()
	maps.Copy(dirty.Sam, safeReq.PostParams.Sam)
	safeReq.PostParams.Mu.RUnlock()
	dirty.Sam[key] = []string{scan}
	safeReq.PostParams = dirty
	res, err := r.SendHTTPRequest(ctx, domain.MakeHTTPRequestFromSafe(&safeReq))
	if err != nil {
		return
	}

	if r.isCommandInjectionVulnerable(res) {
		val, _ := postParams.LoadOrStore(key, []string{})
		postParams.Store(key, append(val.([]string), scan))
	}
}
