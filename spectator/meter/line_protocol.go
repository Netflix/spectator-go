package meter

import (
	"strconv"
	"sync"

	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

// byteBufPool holds reusable byte-slice buffers for building spectatord
// protocol strings (both line-protocol values and spectator IDs), avoiding
// fmt.Sprintf boxing overhead. Using *[]byte so we can preserve capacity
// across calls by resetting to len=0 without clearing the backing array.
var byteBufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 256)
		return &b
	},
}

// writeLineInt writes "symbol:spectatordId:val" to w using a pooled buffer,
// replacing fmt.Sprintf("%s:%s:%d", ...) with zero-alloc integer formatting.
func writeLineInt(w writer.Writer, symbol string, spectatordId string, val int64) {
	bp := byteBufPool.Get().(*[]byte)
	b := append((*bp)[:0], symbol...)
	b = append(b, ':')
	b = append(b, spectatordId...)
	b = append(b, ':')
	b = strconv.AppendInt(b, val, 10)
	w.Write(string(b))
	*bp = b
	byteBufPool.Put(bp)
}

// writeLineFloat writes "symbol:spectatordId:val" to w using a pooled buffer,
// replacing fmt.Sprintf("%s:%s:%f", ...) with zero-alloc float formatting.
func writeLineFloat(w writer.Writer, symbol string, spectatordId string, val float64) {
	bp := byteBufPool.Get().(*[]byte)
	b := append((*bp)[:0], symbol...)
	b = append(b, ':')
	b = append(b, spectatordId...)
	b = append(b, ':')
	b = strconv.AppendFloat(b, val, 'f', 6, 64)
	w.Write(string(b))
	*bp = b
	byteBufPool.Put(bp)
}

// writeLineUint writes "symbol:spectatordId:val" to w using a pooled buffer,
// replacing fmt.Sprintf("%s:%s:%d", ...) with zero-alloc uint formatting.
func writeLineUint(w writer.Writer, symbol string, spectatordId string, val uint64) {
	bp := byteBufPool.Get().(*[]byte)
	b := append((*bp)[:0], symbol...)
	b = append(b, ':')
	b = append(b, spectatordId...)
	b = append(b, ':')
	b = strconv.AppendUint(b, val, 10)
	w.Write(string(b))
	*bp = b
	byteBufPool.Put(bp)
}
