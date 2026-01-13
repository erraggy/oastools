package converter

import "sync"

// Pool capacity (corpus-validated: P75=6,319 refs)
const (
	conversionMapCap    = 8192
	conversionMapMaxCap = 16384
)

var conversionMapPool = sync.Pool{
	New: func() any {
		return make(map[string]string, conversionMapCap)
	},
}

func getConversionMap() map[string]string {
	m := conversionMapPool.Get().(map[string]string)
	clear(m)
	return m
}

func putConversionMap(m map[string]string) {
	if m == nil || len(m) > conversionMapMaxCap {
		return
	}
	conversionMapPool.Put(m)
}
