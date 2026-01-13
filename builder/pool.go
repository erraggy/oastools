package builder

import (
	"sync"

	"github.com/erraggy/oastools/parser"
)

const (
	schemaMapCap      = 8
	pathMapCap        = 4
	operationSliceCap = 8
)

var schemaMapPool = sync.Pool{
	New: func() any {
		return make(map[string]*parser.Schema, schemaMapCap)
	},
}

func getSchemaMap() map[string]*parser.Schema {
	m := schemaMapPool.Get().(map[string]*parser.Schema)
	clear(m)
	return m
}

func putSchemaMap(m map[string]*parser.Schema) {
	if m == nil || len(m) > 128 {
		return
	}
	schemaMapPool.Put(m)
}

var pathMapPool = sync.Pool{
	New: func() any {
		return make(map[string]*parser.PathItem, pathMapCap)
	},
}

func getPathMap() map[string]*parser.PathItem {
	m := pathMapPool.Get().(map[string]*parser.PathItem)
	clear(m)
	return m
}

func putPathMap(m map[string]*parser.PathItem) {
	if m == nil || len(m) > 64 {
		return
	}
	pathMapPool.Put(m)
}
