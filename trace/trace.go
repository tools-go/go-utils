package trace

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/tools-go/go-utils/trace/zaplog"
)

const (
	stackDepth = 2
)

//Trace is a log trace utils wrapped on glog, can be used to trace a http request and its subrequests
type Trace interface {
	// Parent will return the parent trace
	Parent() Trace
	// Name will return the current trace name
	Name() string
	// SetName will set a new name for the Trace object
	SetName(name string)
	// ID will return the current trace id
	ID() string
	// Start will return the current trace start time
	Start() time.Time
	// Duration will return the current trace Duration
	Duration() time.Duration
	// Info will print the args as the info level log
	Info(args ...interface{})
	// Infof will print the args with a format as the info level log
	Infof(format string, args ...interface{})
	// Warn will print the args as the warn level log
	Warn(args ...interface{})
	// Warnf will print the args with a format as the warn level log
	Warnf(format string, args ...interface{})
	// Error will print the args as the error level log
	Error(args ...interface{})
	// Errorf will print the args with a format as the error level log
	Errorf(format string, args ...interface{})
	// Stack will return current stack
	Stack(all ...bool) string
	// String will return a string-serialized trace
	String() string

	Debug(args ...interface{})

	Debugf(format string, args ...interface{})
}

type trace struct {
	parent    Trace
	startTime time.Time
	name      string
	id        string
	head      string
}

//New will create a Trace using a name, identifying the trace process
func New(name string, id ...string) Trace {
	if len(id) > 0 && len(id[0]) > 0 {
		return WithID(name, id[0])
	}
	return WithParent(nil, name)
}

//WithParent will create a Trace use a parent Trace and a identified name
func WithParent(p Trace, name string) Trace {
	t := &trace{
		parent:    p,
		startTime: time.Now(),
		name:      name,
	}

	if p != nil {
		t.id = p.ID()
	} else {
		t.id = zaplog.WithTraceID()
	}

	t.head = t.packHeader()
	return t
}

//WithID will create a Trace with a name and a trace id
func WithID(name string, id string) Trace {
	t := &trace{
		parent:    nil,
		startTime: time.Now(),
		name:      name,
		id:        id,
	}
	t.head = t.packHeader()
	return t
}

func (t *trace) packHeader() string {
	var buffer bytes.Buffer

	buffer.WriteString("tname=[")
	buffer.WriteString(t.Name())
	buffer.WriteString("] ")

	buffer.WriteString("tid=[")
	buffer.WriteString(t.ID())
	buffer.WriteString("] ")

	if t.parent != nil {
		buffer.WriteString("tancestor=[")
		for np := t.parent; np != nil; np = np.Parent() {
			if np != t.parent {
				buffer.WriteString(",")
			}
			buffer.WriteString(np.Name())
		}
		buffer.WriteString("] ")
	}

	buffer.WriteString("tduration=[")

	return buffer.String()
}

func (t *trace) header() string {
	return t.head + strconv.Itoa(int(t.Duration())) + "] "
}

func (t *trace) Parent() Trace {
	return t.parent
}

func (t *trace) Name() string {
	return t.name
}

func (t *trace) SetName(name string) {
	t.name = name
}

func (t *trace) ID() string {
	return t.id
}

func (t *trace) Start() time.Time {
	return t.startTime
}

func (t *trace) Duration() time.Duration {
	// time.Millisecond
	return time.Since(t.startTime) / time.Millisecond
}

// copy this from glog
func Stacks(all bool) []byte {
	n := 10000
	if all {
		n = 100000
	}
	var trace []byte
	for i := 0; i < 5; i++ {
		trace = make([]byte, n)
		nbytes := runtime.Stack(trace, all)
		if nbytes < len(trace) {
			return trace[:nbytes]
		}
		n *= 2
	}
	return trace
}

func (t *trace) String() string {
	return t.header()
}

func (t *trace) Stack(all ...bool) string {
	dumpAll := false
	if len(all) > 0 {
		dumpAll = all[0]
	}
	return string(Stacks(dumpAll))
}

func (t *trace) log(out func(args ...interface{}), args ...interface{}) {
	var newArgs []interface{}
	newArgs = append(newArgs, t.header())
	if len(args) > 0 {
		newArgs = append(newArgs, args...)
	}

	out(newArgs...)
}

func (t *trace) logf(out func(args ...interface{}), format string, args ...interface{}) {
	log := fmt.Sprintf(t.header()+format, args...)
	out(log)
	//out(t.header()+format, stackDepth, args...)
}

func (t *trace) Info(args ...interface{}) {
	t.log(zaplog.Logger.Info, args...)
}

func (t *trace) Infof(format string, args ...interface{}) {
	t.logf(zaplog.Logger.Info, format, args...)
}

func (t *trace) Warn(args ...interface{}) {
	t.log(zaplog.Logger.Warn, args...)
}

func (t *trace) Warnf(format string, args ...interface{}) {
	t.logf(zaplog.Logger.Warn, format, args...)
}

func (t *trace) Error(args ...interface{}) {
	t.log(zaplog.Logger.Error, args...)
}

func (t *trace) Errorf(format string, args ...interface{}) {
	t.logf(zaplog.Logger.Error, format, args...)
}

func (t *trace) Debug(args ...interface{}) {
	t.log(zaplog.Logger.Debug, args...)
}

func (t *trace) Debugf(format string, args ...interface{}) {
	t.logf(zaplog.Logger.Debug, format, args...)
}
