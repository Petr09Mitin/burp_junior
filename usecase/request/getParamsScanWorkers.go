package request

import (
	"context"
	"maps"
	"sync"

	"github.com/burp_junior/domain"
)

func (r *RequestService) scanGetParamsWorker(ctx context.Context, wg *sync.WaitGroup, req *domain.SafeHTTPRequest, ci SafeInjections, getParams *sync.Map) {
	defer wg.Done()

	getParamsWg := &sync.WaitGroup{}
	getParamsWg.Add(len(req.GetParams.Sam))

	req.GetParams.Mu.RLock()
	for key := range req.GetParams.Sam {
		go r.scanGetParamWorker(ctx, getParamsWg, req, key, ci, getParams)
	}

	req.GetParams.Mu.RUnlock()

	getParamsWg.Wait()
}

func (r *RequestService) scanGetParamWorker(ctx context.Context, wg *sync.WaitGroup, safeReq *domain.SafeHTTPRequest, key string, ci SafeInjections, getParams *sync.Map) {
	defer wg.Done()
	getParamsScansWg := &sync.WaitGroup{}
	ci.mu.RLock()
	getParamsScansWg.Add(len(ci.ci))

	for _, scan := range ci.ci {
		go r.sendGetParamScan(ctx, getParamsScansWg, *safeReq, key, scan, getParams)
	}
	ci.mu.RUnlock()

	getParamsScansWg.Wait()
}

func (r *RequestService) sendGetParamScan(ctx context.Context, wg *sync.WaitGroup, safeReq domain.SafeHTTPRequest, key string, scan string, getParams *sync.Map) {
	defer wg.Done()
	dirty := domain.SafeStringArrMap{
		Sam: map[string][]string{},
		Mu:  &sync.RWMutex{},
	}
	safeReq.GetParams.Mu.RLock()
	maps.Copy(dirty.Sam, safeReq.GetParams.Sam)
	safeReq.GetParams.Mu.RUnlock()
	dirty.Sam[key] = []string{scan}
	safeReq.GetParams = dirty
	res, err := r.SendHTTPRequest(ctx, domain.MakeHTTPRequestFromSafe(&safeReq))
	if err != nil {
		return
	}

	if r.isCommandInjectionVulnerable(res) {
		val, _ := getParams.LoadOrStore(key, []string{})
		getParams.Store(key, append(val.([]string), scan))
	}
}
