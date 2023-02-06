package datamap

import (
	"context"
	"strings"

	"github.com/snabble/go-logging/v2/tracex/propagation"
)

type Propagator struct {
	propagation propagation.TraceContext
}

func NewPropagator() *Propagator {
	return &Propagator{propagation: propagation.TraceContext{}}
}

// Container contains a datamap
type Container interface {
	GetDataMap() map[string]string
}

func (p *Propagator) Extract(ctx context.Context, owner Container) context.Context {
	return p.propagation.Extract(ctx, dataMap(owner.GetDataMap()))
}

func (p *Propagator) Inject(ctx context.Context) map[string]string {
	carrier := dataMap{}
	p.propagation.Inject(ctx, carrier)
	return carrier
}

const tracingPrefix = "__tracing__"

type dataMap map[string]string

func (dm dataMap) Get(key string) string {
	value, ok := dm[tracingPrefix+key]
	if !ok {
		return ""
	}
	return value
}

func (dm dataMap) Set(key string, value string) {
	dm[tracingPrefix+key] = value
}

func (dm dataMap) Keys() []string {
	keys := make([]string, 0, len(dm))
	for key := range dm {
		if !strings.HasPrefix(key, tracingPrefix) {
			continue
		}
		keys = append(keys, strings.TrimPrefix(key, tracingPrefix))
	}
	return keys
}
