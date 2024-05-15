package trace

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

const (
	X_REQUEST_HEADER_RID = "X-Request-Id"
)

var rndPool = sync.Pool{
	New: func() interface{} {
		return rand.New(rand.NewSource(time.Now().UnixNano()))
	},
}

type Trace struct {
	TraceId     string
	SpanId      string
	HintCode    int64
	HintContent HintContent
}

type HintContent struct {
	Sample HintSampling `json:"Sample"`
}

type HintSampling struct {
	Rate int   `json:"Rate"`
	Code int64 `json:"Code"`
}

var ip string

func NewSpanId(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if sc.SpanID().IsValid() {
		return sc.SpanID().String()
	}

	rnd := rndPool.Get().(*rand.Rand)
	defer rndPool.Put(rnd)

	return fmt.Sprintf("%d", rnd.Int63())
}

// GetTraceFromHeader 从http request中获取header中的trace信息
func GetTraceFromHeader(request *http.Request) (t *Trace) {
	t = &Trace{}
	sc := trace.SpanContextFromContext(request.Context())
	if sc.TraceID().IsValid() {
		t.TraceId = sc.TraceID().String()
		t.SpanId = sc.SpanID().String()
	}

	return t
}

// GenTraceId 生成traceid
func GenTraceId(ctx context.Context) (traceId string) {
	sc := trace.SpanContextFromContext(ctx)
	if sc.TraceID().IsValid() {
		return sc.TraceID().String()
	}

	if ip == "" {
		ip = GetIp()
	}
	return calculateTraceId(ip)
}

func calculateTraceId(ip string) (traceId string) {
	now := time.Now()
	timestamp := uint32(now.Unix())
	timeNano := now.UnixNano()
	pid := os.Getpid()
	b := bytes.Buffer{}
	rnd := rndPool.Get().(*rand.Rand)
	defer rndPool.Put(rnd)

	b.WriteString(hex.EncodeToString(net.ParseIP(ip).To4()))
	b.WriteString(fmt.Sprintf("%x", timestamp&0xffffffff))
	b.WriteString(fmt.Sprintf("%04x", timeNano&0xffff))
	b.WriteString(fmt.Sprintf("%04x", pid&0xffff))
	b.WriteString(fmt.Sprintf("%06x", rnd.Int31n(1<<24)))
	b.WriteString("b0")

	return b.String()
}

func (l *Logger) GetTrace(ctx context.Context) *Trace {
	if l.Trace == nil {
		l.Trace = &Trace{}
	}
	t := l.Trace
	if len(t.TraceId) <= 0 {
		t.TraceId = GenTraceId(ctx)
	}
	if len(t.SpanId) <= 0 {
		t.SpanId = NewSpanId(ctx)
	}
	return t
}

func (l *Logger) ResetTrace(ctx context.Context) *Trace {
	l.Trace = &Trace{}
	return l.GetTrace(ctx)
}

func (l *Logger) ParseTrace(req *http.Request) {
	l.Trace = GetTraceFromHeader(req)
}

func (l *Logger) SetTraceToWriter(ctx context.Context, rw http.ResponseWriter) {
	rw.Header().Set(X_REQUEST_HEADER_RID, l.GetTrace(ctx).GetTraceId())
}

func (t *Trace) GetTraceId() string {
	return t.TraceId
}

func (t *Trace) GetSpanId() string {
	return t.SpanId
}

func (t *Trace) GetHintCode() string {
	return fmt.Sprint(t.HintCode)
}

func (t *Trace) GetHintContent() string {
	if t.HintContent != (HintContent{}) {
		if hc, err := json.Marshal(t.HintContent); err == nil {
			return string(hc)
		}
	}
	return ""
}

func GetIntranetIpv4() (ip string, err error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1", err
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				break
			}
		}
	}
	return ip, err
}

func GetIp() string {
	ip, _ := GetIntranetIpv4()
	return ip
}
