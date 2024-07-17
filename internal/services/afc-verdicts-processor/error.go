package afcverdictsprocessor

import "errors"

func isRetriable(err error) bool {
	var r interface{ Retriable() bool }
	return errors.As(err, &r) && r.Retriable()
}

type retriableError struct {
	error
}

func newRetriableError(err error) *retriableError {
	return &retriableError{error: err}
}

func (e *retriableError) Unwrap() error   { return e.error }
func (e *retriableError) Retriable() bool { return true }

func extractVerdict(err error) (verdict, bool) {
	var verdictProvider interface{ verdict() verdict }
	if errors.As(err, &verdictProvider) {
		return verdictProvider.verdict(), true
	}
	return verdict{}, false
}

type withVerdictError struct {
	error
	v verdict
}

func attachVerdict(err error, v verdict) *withVerdictError {
	return &withVerdictError{error: err, v: v}
}

func (e *withVerdictError) Unwrap() error    { return e.error }
func (e *withVerdictError) verdict() verdict { return e.v }
