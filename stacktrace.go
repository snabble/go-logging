// Copyright (c) 2019 Sentry (https://sentry.io) and individual contributors.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:
//
// * Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES,
// INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY,
// WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE
// USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package logging

import (
	"go/build"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

const unknown string = "unknown"

// Stacktrace holds information about the frames of the stack.
type Stacktrace struct {
	Frames        []Frame `json:"frames,omitempty"`
	FramesOmitted []uint  `json:"frames_omitted,omitempty"`
}

// NewStacktrace creates a stacktrace using runtime.Callers.
func NewStacktrace() *Stacktrace {
	pcs := make([]uintptr, 100)
	n := runtime.Callers(1, pcs)

	if n == 0 {
		return nil
	}

	frames := extractFrames(pcs[:n])
	frames = filterFrames(frames)

	stacktrace := Stacktrace{
		Frames: frames,
	}

	return &stacktrace
}

// ExtractStacktrace creates a new Stacktrace based on the given error.
func ExtractStacktrace(err error) *Stacktrace {
	method := extractReflectedStacktraceMethod(err)

	var pcs []uintptr

	if method.IsValid() {
		pcs = extractPcs(method)
	} else {
		pcs = extractXErrorsPC(err)
	}

	if len(pcs) == 0 {
		return nil
	}

	frames := extractFrames(pcs)
	frames = filterFrames(frames)

	stacktrace := Stacktrace{
		Frames: frames,
	}

	return &stacktrace
}

func extractReflectedStacktraceMethod(err error) reflect.Value {
	var method reflect.Value

	// https://github.com/pingcap/errors
	methodGetStackTracer := reflect.ValueOf(err).MethodByName("GetStackTracer")
	// https://github.com/pkg/errors
	methodStackTrace := reflect.ValueOf(err).MethodByName("StackTrace")
	// https://github.com/go-errors/errors
	methodStackFrames := reflect.ValueOf(err).MethodByName("StackFrames")

	if methodGetStackTracer.IsValid() {
		stacktracer := methodGetStackTracer.Call(make([]reflect.Value, 0))[0]
		stacktracerStackTrace := reflect.ValueOf(stacktracer).MethodByName("StackTrace")

		if stacktracerStackTrace.IsValid() {
			method = stacktracerStackTrace
		}
	}

	if methodStackTrace.IsValid() {
		method = methodStackTrace
	}

	if methodStackFrames.IsValid() {
		method = methodStackFrames
	}

	return method
}

func extractPcs(method reflect.Value) []uintptr {
	var pcs []uintptr

	stacktrace := method.Call(make([]reflect.Value, 0))[0]

	if stacktrace.Kind() != reflect.Slice {
		return nil
	}

	for i := 0; i < stacktrace.Len(); i++ {
		pc := stacktrace.Index(i)

		if pc.Kind() == reflect.Uintptr {
			pcs = append(pcs, uintptr(pc.Uint()))
			continue
		}

		if pc.Kind() == reflect.Struct {
			field := pc.FieldByName("ProgramCounter")
			if field.IsValid() && field.Kind() == reflect.Uintptr {
				pcs = append(pcs, uintptr(field.Uint()))
				continue
			}
		}
	}

	return pcs
}

// extractXErrorsPC extracts program counters from error values compatible with
// the error types from golang.org/x/xerrors.
//
// It returns nil if err is not compatible with errors from that package or if
// no program counters are stored in err.
func extractXErrorsPC(err error) []uintptr {
	// This implementation uses the reflect package to avoid a hard dependency
	// on third-party packages.

	// We don't know if err matches the expected type. For simplicity, instead
	// of trying to account for all possible ways things can go wrong, some
	// assumptions are made and if they are violated the code will panic. We
	// recover from any panic and ignore it, returning nil.
	//nolint: errcheck
	defer func() { recover() }()

	field := reflect.ValueOf(err).Elem().FieldByName("frame") // type Frame struct{ frames [3]uintptr }
	field = field.FieldByName("frames")
	field = field.Slice(1, field.Len()) // drop first pc pointing to xerrors.New
	pc := make([]uintptr, field.Len())
	for i := 0; i < field.Len(); i++ {
		pc[i] = uintptr(field.Index(i).Uint())
	}
	return pc
}

// Frame represents a function call and it's metadata. Frames are associated
// with a Stacktrace.
type Frame struct {
	Function    string                 `json:"function,omitempty"`
	Symbol      string                 `json:"symbol,omitempty"`
	Module      string                 `json:"module,omitempty"`
	Package     string                 `json:"package,omitempty"`
	Filename    string                 `json:"filename,omitempty"`
	AbsPath     string                 `json:"abs_path,omitempty"`
	Lineno      int                    `json:"lineno,omitempty"`
	Colno       int                    `json:"colno,omitempty"`
	PreContext  []string               `json:"pre_context,omitempty"`
	ContextLine string                 `json:"context_line,omitempty"`
	PostContext []string               `json:"post_context,omitempty"`
	InApp       bool                   `json:"in_app,omitempty"`
	Vars        map[string]interface{} `json:"vars,omitempty"`
}

// NewFrame assembles a stacktrace frame out of runtime.Frame.
func NewFrame(f runtime.Frame) Frame {
	var abspath, relpath string
	// NOTE: f.File paths historically use forward slash as path separator even
	// on Windows, though this is not yet documented, see
	// https://golang.org/issues/3335. In any case, filepath.IsAbs can work with
	// paths with either slash or backslash on Windows.
	switch {
	case f.File == "":
		relpath = unknown
		// Leave abspath as the empty string to be omitted when serializing
		// event as JSON.
		abspath = ""
	case filepath.IsAbs(f.File):
		abspath = f.File
		// TODO: in the general case, it is not trivial to come up with a
		// "project relative" path with the data we have in run time.
		// We shall not use filepath.Base because it creates ambiguous paths and
		// affects the "Suspect Commits" feature.
		// For now, leave relpath empty to be omitted when serializing the event
		// as JSON. Improve this later.
		relpath = ""
	default:
		// f.File is a relative path. This may happen when the binary is built
		// with the -trimpath flag.
		relpath = f.File
		// Omit abspath when serializing the event as JSON.
		abspath = ""
	}

	function := f.Function
	var pkg string

	if function != "" {
		pkg, function = splitQualifiedFunctionName(function)
	}

	frame := Frame{
		AbsPath:  abspath,
		Filename: relpath,
		Lineno:   f.Line,
		Module:   pkg,
		Function: function,
	}

	frame.InApp = isInAppFrame(frame)

	return frame
}

// splitQualifiedFunctionName splits a package path-qualified function name into
// package name and function name. Such qualified names are found in
// runtime.Frame.Function values.
func splitQualifiedFunctionName(name string) (pkg string, fun string) {
	pkg = packageName(name)
	fun = strings.TrimPrefix(name, pkg+".")
	return
}

func extractFrames(pcs []uintptr) []Frame {
	var frames []Frame
	callersFrames := runtime.CallersFrames(pcs)

	for {
		callerFrame, more := callersFrames.Next()

		frames = append([]Frame{
			NewFrame(callerFrame),
		}, frames...)

		if !more {
			break
		}
	}

	return frames
}

// filterFrames filters out stack frames that are not meant to be reported to
// Sentry. Those are frames internal to the SDK or Go.
func filterFrames(frames []Frame) []Frame {
	if len(frames) == 0 {
		return nil
	}

	filteredFrames := make([]Frame, 0, len(frames))

	for _, frame := range frames {
		// Skip Go internal frames.
		if frame.Module == "runtime" || frame.Module == "testing" {
			continue
		}
		// Skip snabble internal frames.
		//if strings.HasPrefix(frame.Module, "github.com/snabble/go-logging") {
		//	continue
		//}
		filteredFrames = append(filteredFrames, frame)
	}

	return filteredFrames
}

func isInAppFrame(frame Frame) bool {
	if strings.HasPrefix(frame.AbsPath, build.Default.GOROOT) ||
		strings.Contains(frame.Module, "vendor") ||
		strings.Contains(frame.Module, "third_party") {
		return false
	}

	return true
}

// packageName returns the package part of the symbol name, or the empty string
// if there is none.
// It replicates https://golang.org/pkg/debug/gosym/#Sym.PackageName, avoiding a
// dependency on debug/gosym.
func packageName(name string) string {
	// A prefix of "type." and "go." is a compiler-generated symbol that doesn't belong to any package.
	// See variable reservedimports in cmd/compile/internal/gc/subr.go
	if strings.HasPrefix(name, "go.") || strings.HasPrefix(name, "type.") {
		return ""
	}

	pathend := strings.LastIndex(name, "/")
	if pathend < 0 {
		pathend = 0
	}

	if i := strings.Index(name[pathend:], "."); i != -1 {
		return name[:pathend+i]
	}
	return ""
}
