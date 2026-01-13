package httpvalidator

import "sync"

// Pool capacities (corpus-validated)
const (
	requestResultErrorsCap   = 8
	requestResultWarningsCap = 4
)

var requestResultPool = sync.Pool{
	New: func() any {
		return &RequestValidationResult{
			Errors:       make([]ValidationError, 0, requestResultErrorsCap),
			Warnings:     make([]ValidationError, 0, requestResultWarningsCap),
			PathParams:   make(map[string]any),
			QueryParams:  make(map[string]any),
			HeaderParams: make(map[string]any),
			CookieParams: make(map[string]any),
		}
	},
}

// getRequestResult retrieves a RequestValidationResult from the pool and resets it.
func getRequestResult() *RequestValidationResult {
	r := requestResultPool.Get().(*RequestValidationResult)
	r.reset()
	return r
}

// putRequestResult returns a RequestValidationResult to the pool.
func putRequestResult(r *RequestValidationResult) {
	if r == nil {
		return
	}
	requestResultPool.Put(r)
}

var responseResultPool = sync.Pool{
	New: func() any {
		return &ResponseValidationResult{
			Errors:   make([]ValidationError, 0, requestResultErrorsCap),
			Warnings: make([]ValidationError, 0, requestResultWarningsCap),
		}
	},
}

// getResponseResult retrieves a ResponseValidationResult from the pool and resets it.
func getResponseResult() *ResponseValidationResult {
	r := responseResultPool.Get().(*ResponseValidationResult)
	r.reset()
	return r
}

// putResponseResult returns a ResponseValidationResult to the pool.
func putResponseResult(r *ResponseValidationResult) {
	if r == nil {
		return
	}
	responseResultPool.Put(r)
}
