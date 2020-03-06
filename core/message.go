package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/peak/s5cmd/log"
	"github.com/peak/s5cmd/objurl"
	"github.com/peak/s5cmd/storage"
)

// InfoMessage is a generic message structure for successful operations.
type InfoMessage struct {
	Operation   string            `json:"operation"`
	Success     bool              `json:"success"`
	Source      *objurl.ObjectURL `json:"source"`
	Destination *objurl.ObjectURL `json:"destination,omitempty"`
	Object      *storage.Object   `json:"object,omitempty"`
}

// String is the string representation of InfoMessage.
func (i InfoMessage) String() string {
	return fmt.Sprintf("%v %v", i.Operation, i.Source)
}

// JSON is the JSON representation of InfoMessage.
func (i InfoMessage) JSON() string {
	i.Success = true
	return jsonMarshal(i)
}

// ErrorMessage is a generic message structure for unsuccessful operations.
type ErrorMessage struct {
	Operation string `json:"operation,omitempty"`
	Job       string `json:"job,omitempty"`
	Err       string `json:"error"`

	format string
}

// String is the string representation of ErrorMessage.
func (e ErrorMessage) String() string {
	if e.Job == "" {
		return fmt.Sprintf(e.format, e.Err)
	}
	return fmt.Sprintf(e.format, e.Job, e.Err)
}

// JSON is the JSON representation of ErrorMessage.
func (e ErrorMessage) JSON() string {
	return jsonMarshal(e)
}

// newErrorMessage creates new ErrorMessage.
func newErrorMessage(job, operation string, err error, format string) ErrorMessage {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	return ErrorMessage{
		Operation: operation,
		Job:       job,
		Err:       cleanupSpaces(errStr),
		format:    format,
	}
}

// printWarning is the helper function to log warning messages.
func printWarning(job, operation string, err error) {
	format := "%q (%v)"
	msg := newErrorMessage(job, operation, err, format)
	log.Warning(msg)
}

// printError is the helper function to log error messages.
func printError(job, operation string, err error) {
	format := "%q %v"
	msg := newErrorMessage(job, operation, err, format)
	log.Error(msg)
}

// DebugMessage is a generic message structure for debugging logs.
type DebugMessage struct {
	Content string `json:"content"`
}

// String is the string representation of DebugMessage.
func (d DebugMessage) String() string {
	return d.Content
}

// JSON is the JSON representation of DebugMessage.
func (d DebugMessage) JSON() string {
	return jsonMarshal(d)
}

// printDebug is the helper function to log debug messages.
func printDebug(format string, args ...interface{}) {
	content := fmt.Sprintf(format, args...)
	msg := DebugMessage{Content: content}
	log.Debug(msg)
}

// cleanupSpaces converts multiline messages into
// a single line.
func cleanupSpaces(s string) string {
	s = strings.Replace(s, "\n", " ", -1)
	s = strings.Replace(s, "\t", " ", -1)
	s = strings.Replace(s, "  ", " ", -1)
	s = strings.TrimSpace(s)
	return s
}

// jsonMarshall is a helper function for creating JSON-encoded strings.
func jsonMarshal(v interface{}) string {
	bytes, _ := json.Marshal(v)
	return string(bytes)
}

// FIXME(ig): move to a proper place
func isCancelationError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) {
		return true
	}

	if storage.IsCancelationError(err) {
		return true
	}

	merr, ok := err.(*multierror.Error)
	if !ok {
		return false
	}

	for _, err := range merr.Errors {
		if isCancelationError(err) {
			return true
		}
	}

	return false
}
