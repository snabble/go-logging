package logging

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// AccessLogCookiesBlacklist The of cookies which should not be logged
var AccessLogCookiesBlacklist = []string{}

var LifecycleEnvVars = []string{"BUILD_NUMBER", "BUILD_HASH", "BUILD_DATE"}

func init() {
	Set("info", false)
}

// Set creates a new Logger with the matching specification
func Set(level string, textLogging bool) error {
	l, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}

	logger := logrus.New()
	if textLogging {
		logger.Formatter = &logrus.TextFormatter{DisableColors: true}
	} else {
		logger.Formatter = &LogstashFormatter{TimestampFormat: time.RFC3339Nano}
	}
	logger.Level = l
	Log = &Logger{logger}
	return nil
}

// Access logs an access entry with call duration and status code
func Access(r *http.Request, start time.Time, statusCode int) {
	e := access(r, start, statusCode, nil)

	var msg string
	if len(r.URL.RawQuery) == 0 {
		msg = fmt.Sprintf("%v ->%v %v", statusCode, r.Method, r.URL.Path)
	} else {
		msg = fmt.Sprintf("%v ->%v %v?%s", statusCode, r.Method, r.URL.Path, r.URL.RawQuery)
	}

	if statusCode >= 200 && statusCode <= 399 {
		e.Info(msg)
	} else if statusCode >= 400 && statusCode <= 499 {
		e.Warn(msg)
	} else {
		e.Error(msg)
	}
}

// AccessError logs an error while accessing
func AccessError(r *http.Request, start time.Time, err error) {
	e := access(r, start, 0, err)
	e.Errorf("ERROR ->%v %v", r.Method, r.URL.Path)
}

func access(r *http.Request, start time.Time, statusCode int, err error) *Entry {
	url := r.URL.Path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}
	fields := logrus.Fields{
		"type":       "access",
		"@timestamp": start,
		"remote_ip":  getRemoteIp(r),
		"host":       r.Host,
		"url":        url,
		"method":     r.Method,
		"proto":      r.Proto,
		"duration":   time.Since(start).Nanoseconds() / 1000000,
		"User_Agent": r.Header.Get("User-Agent"),
	}

	if statusCode != 0 {
		fields["response_status"] = statusCode
	}

	if err != nil {
		fields[logrus.ErrorKey] = err.Error()
	}

	setCorrelationIds(fields, r.Header)

	cookies := map[string]string{}
	for _, c := range r.Cookies() {
		if !contains(AccessLogCookiesBlacklist, c.Name) {
			cookies[c.Name] = c.Value
		}
	}
	if len(cookies) > 0 {
		fields["cookies"] = cookies
	}

	return Log.WithFields(fields)
}

// Call logs the result of an outgoing call
func Call(r *http.Request, resp *http.Response, start time.Time, err error) {
	fields := fieldsForCall(r, resp, start, err)
	logCall(fields, r, resp, err)
}

// FlakyCall logs the result of an outgoing call and marks it as flaky
func FlakyCall(r *http.Request, resp *http.Response, start time.Time, err error) {
	fields := fieldsForCall(r, resp, start, err)
	fields["flaky"] = true
	logCall(fields, r, resp, err)
}

func fieldsForCall(r *http.Request, resp *http.Response, start time.Time, err error) logrus.Fields {
	url := r.URL.Path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}
	fields := logrus.Fields{
		"type":       "call",
		"@timestamp": start,
		"host":       r.Host,
		"url":        url,
		"full_url":   r.URL.String(),
		"method":     r.Method,
		"duration":   time.Since(start).Nanoseconds() / 1000000,
	}

	setCorrelationIds(fields, r.Header)

	if err != nil {
		fields[logrus.ErrorKey] = err.Error()
	}

	if resp != nil {
		fields["response_status"] = resp.StatusCode
		fields["content_type"] = resp.Header.Get("Content-Type")
	}

	return fields
}

func logCall(fields logrus.Fields, r *http.Request, resp *http.Response, err error) {
	if err != nil {
		Log.WithFields(fields).Error(err)
		return
	}

	if resp != nil {
		e := Log.WithFields(fields)
		msg := fmt.Sprintf("%v %v-> %v", resp.StatusCode, r.Method, r.URL.String())

		if resp.StatusCode >= 200 && resp.StatusCode <= 399 {
			e.Info(msg)
		} else if resp.StatusCode >= 400 && resp.StatusCode <= 499 {
			e.Warn(msg)
		} else {
			e.Error(msg)
		}
		return
	}

	Log.WithFields(fields).Warn("call, but no response given")
}

// Cacheinfo logs the hit information a accessing a resource
func Cacheinfo(url string, hit bool) {
	var msg string
	if hit {
		msg = fmt.Sprintf("cache hit: %v", url)
	} else {
		msg = fmt.Sprintf("cache miss: %v", url)
	}
	Log.WithFields(
		logrus.Fields{
			"type": "cacheinfo",
			"url":  url,
			"hit":  hit,
		}).
		Debug(msg)
}

// Application Return a log entry for application logs,
// prefilled with the correlation ids out of the supplied request.
func Application(h http.Header) *Entry {
	fields := logrus.Fields{
		"type": "application",
	}
	setCorrelationIds(fields, h)
	return Log.WithFields(fields)
}

// LifecycleStart logs the start of an application
// with the configuration struct or map as parameter.
func LifecycleStart(appName string, args interface{}) {
	fields := logrus.Fields{}

	jsonString, err := json.Marshal(args)
	if err == nil {
		err := json.Unmarshal(jsonString, &fields)
		if err != nil {
			fields["parse_error"] = err.Error()
		}
	}
	fields["type"] = "lifecycle"
	fields["event"] = "start"
	for _, env := range LifecycleEnvVars {
		if os.Getenv(env) != "" {
			fields[strings.ToLower(env)] = os.Getenv(env)
		}
	}

	Log.WithFields(fields).Infof("starting application: %v", appName)
}

// LifecycleStop logs the request to stop an application
func LifecycleStop(appName string, signal os.Signal, err error) {
	fields := logrus.Fields{
		"type":  "lifecycle",
		"event": "stop",
	}
	if signal != nil {
		fields["signal"] = signal.String()
	}

	if os.Getenv("BUILD_NUMBER") != "" {
		fields["build_number"] = os.Getenv("BUILD_NUMBER")
	}

	if err != nil {
		Log.WithFields(fields).
			WithError(err).
			Errorf("stopping application: %v (%v)", appName, err)
	} else {
		Log.WithFields(fields).Infof("stopping application: %v (%v)", appName, signal)
	}
}

// LifecycleStoped logs the stop of an application
// Deprecated: Typo in name LifecycleStoped, please use LifecycleStopped instead.
func LifecycleStoped(appName string, err error) {
	logApplicationLifecycleEvent(appName, "stoped", err)
}

// LifecycleStopped logs the stop of an application
func LifecycleStopped(appName string, err error) {
	logApplicationLifecycleEvent(appName, "stopped", err)
}

func logApplicationLifecycleEvent(appName string, eventName string, err error) {
	fields := logrus.Fields{
		"type":  "lifecycle",
		"event": eventName,
	}

	if os.Getenv("BUILD_NUMBER") != "" {
		fields["build_number"] = os.Getenv("BUILD_NUMBER")
	}

	if err != nil {
		Log.WithFields(fields).
			WithError(err).
			Errorf("stopping application: %v (%v)", appName, err)
	} else {
		Log.WithFields(fields).Infof("application %s: %v", eventName, appName)
	}
}

// ServerClosed logs the clos of an server
func ServerClosed(appName string) {
	fields := logrus.Fields{
		"type":  "application",
		"event": "stop",
	}

	if os.Getenv("BUILD_NUMBER") != "" {
		fields["build_number"] = os.Getenv("BUILD_NUMBER")
	}

	Log.WithFields(fields).Infof("http server was closed: %v", appName)
}

func getRemoteIp(r *http.Request) string {
	if r.Header.Get("X-Cluster-Client-Ip") != "" {
		return r.Header.Get("X-Cluster-Client-Ip")
	}
	if r.Header.Get("X-Real-Ip") != "" {
		return r.Header.Get("X-Real-Ip")
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func setCorrelationIds(fields logrus.Fields, h http.Header) {
	correlationId := GetCorrelationId(h)
	if correlationId != "" {
		fields["correlation_id"] = correlationId
	}

	userCorrelationId := GetUserCorrelationId(h)
	if userCorrelationId != "" {
		fields["user_correlation_id"] = userCorrelationId
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
