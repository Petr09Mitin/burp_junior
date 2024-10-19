package request

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"github.com/burp_junior/domain"
)

func (r *RequestService) scanCookiesWorker(ctx context.Context, wg *sync.WaitGroup, req *domain.SafeHTTPRequest, ci SafeInjections, cookies *sync.Map) {
	defer wg.Done()

	cookiesWg := &sync.WaitGroup{}
	cookiesWg.Add(len(req.Cookies.Sm))

	req.Cookies.Mu.RLock()
	for key := range req.Cookies.Sm {
		go r.scanCookieWorker(ctx, cookiesWg, req, key, ci, cookies)
	}

	req.Cookies.Mu.RUnlock()

	cookiesWg.Wait()
}

func (r *RequestService) scanCookieWorker(ctx context.Context, wg *sync.WaitGroup, safeReq *domain.SafeHTTPRequest, key string, ci SafeInjections, cookies *sync.Map) {
	defer wg.Done()
	cookiesScansWg := &sync.WaitGroup{}
	ci.mu.RLock()
	cookiesScansWg.Add(len(ci.ci))

	for _, scan := range ci.ci {
		go r.sendCookieScan(ctx, cookiesScansWg, *safeReq, key, scan, cookies)
	}
	ci.mu.RUnlock()

	cookiesScansWg.Wait()
}

func (r *RequestService) sendCookieScan(ctx context.Context, wg *sync.WaitGroup, safeReq domain.SafeHTTPRequest, key string, scan string, cookies *sync.Map) {
	defer wg.Done()
	dirty := domain.SafeStringMap{
		Sm: map[string]string{},
		Mu: &sync.RWMutex{},
	}
	safeReq.Cookies.Mu.RLock()
	maps.Copy(dirty.Sm, safeReq.Cookies.Sm)
	safeReq.Cookies.Mu.RUnlock()
	dirty.Sm[key] = fmt.Sprintf("%s=%s", key, scan)
	safeReq.Cookies = dirty
	res, err := r.SendHTTPRequest(ctx, domain.MakeHTTPRequestFromSafe(&safeReq))
	if err != nil {
		return
	}

	if r.isCommandInjectionVulnerable(res) {
		val, _ := cookies.LoadOrStore(key, []string{})
		cookies.Store(key, append(val.([]string), scan))
	}
}
