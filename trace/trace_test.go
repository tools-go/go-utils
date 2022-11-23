// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package trace_test

import (
	"context"
	"flag"
	"testing"

	"github.com/tools-go/go-utils/trace"
)

func TestTrace(t *testing.T) {
	flag.Parse()
	t1 := trace.New("t1")
	t1.Info("=====t1 inflo-213092980")
	t1.Debugf("helllow %s ", "worlad")
	tr := trace.GetTraceFromContext(context.TODO())
	tr.Infof("==t2 with context")
	tr.V(3).Debug("with v debug")
}

//func TestTraceHandler(t *testing.T) {
//	ts := httptest.NewServer(trace.Handler("httptest", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		var tracer trace.Trace
//		if tracer = trace.GetTraceFromRequest(r); tracer == nil {
//			tracer = trace.New("internal-test")
//		}
//		tracer.Info("process start...")
//		defer tracer.Info("process end...")
//
//		tracer.Info("hello test!")
//		fmt.Fprintln(w, `hello test!`)
//	})))
//
//	res, err := http.Get(ts.URL)
//	if err != nil {
//		t.Fatal("get url failed:", err)
//	}
//	defer res.Body.Close()
//
//	ts.Close()
//}
