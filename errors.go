package pgconn

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

// SafeToRetry checks if the err is guaranteed to have occurred before sending any data to the server.

// SafeToRetry 检查错误是否保证发生在发送任何数据到服务器之前。
func SafeToRetry(err error) bool {
	if e, ok := err.(interface{ SafeToRetry() bool }); ok {
		return e.SafeToRetry()
	}
	return false
}

// Timeout checks if err was was caused by a timeout. To be specific, it is true if err was caused within pgconn by a
// context.Canceled, context.DeadlineExceeded or an implementer of net.Error where Timeout() is true.

// Timeout 检查错误是否由超时引起。具体来说，如果错误是在 pgconn 内部由 context.Canceled、context.DeadlineExceeded 或 net.Error 的
// 实现(其中 Timeout() 为 true) ，则此检查结果为 true。
func Timeout(err error) bool {
	var timeoutErr *errTimeout
	return errors.As(err, &timeoutErr)
}

// PgError represents an error reported by the PostgreSQL server. See
// http://www.postgresql.org/docs/11/static/protocol-error-fields.html for
// detailed field description.

// PgError 代表 PostgreSQL 服务器报告的错误。有关详细的字段描述，请参阅
// http://www.postgresql.org/docs/11/static/protocol-error-fields.html。
type PgError struct {
	Severity         string
	Code             string
	Message          string
	Detail           string
	Hint             string
	Position         int32
	InternalPosition int32
	InternalQuery    string
	Where            string
	SchemaName       string
	TableName        string
	ColumnName       string
	DataTypeName     string
	ConstraintName   string
	File             string
	Line             int32
	Routine          string
}

func (pe *PgError) Error() string {
	return pe.Severity + ": " + pe.Message + " (SQLSTATE " + pe.Code + ")"
}

// SQLState returns the SQLState of the error.
func (pe *PgError) SQLState() string {
	return pe.Code
}

type connectError struct {
	config *Config
	msg    string
	err    error
}

func (e *connectError) Error() string {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "failed to connect to `host=%s user=%s database=%s`: %s", e.config.Host, e.config.User, e.config.Database, e.msg)
	if e.err != nil {
		fmt.Fprintf(sb, " (%s)", e.err.Error())
	}
	return sb.String()
}

func (e *connectError) Unwrap() error {
	return e.err
}

type connLockError struct {
	status string
}

func (e *connLockError) SafeToRetry() bool {
	return true // a lock failure by definition happens before the connection is used.
}

func (e *connLockError) Error() string {
	return e.status
}

type parseConfigError struct {
	connString string
	msg        string
	err        error
}

func (e *parseConfigError) Error() string {
	connString := redactPW(e.connString)
	if e.err == nil {
		return fmt.Sprintf("cannot parse `%s`: %s", connString, e.msg)
	}
	return fmt.Sprintf("cannot parse `%s`: %s (%s)", connString, e.msg, e.err.Error())
}

func (e *parseConfigError) Unwrap() error {
	return e.err
}

// preferContextOverNetTimeoutError returns ctx.Err() if ctx.Err() is present and err is a net.Error with Timeout() ==
// true. Otherwise returns err.

// preferContextOverNetTimeoutError 返回 ctx.Err()，如果 ctx.Err() 存在并且 err 是一个 net.Error 且 Timeout() == true。
// 否则返回 err。
func preferContextOverNetTimeoutError(ctx context.Context, err error) error {
	if err, ok := err.(net.Error); ok && err.Timeout() && ctx.Err() != nil {
		return &errTimeout{err: ctx.Err()}
	}
	return err
}

type pgconnError struct {
	msg         string
	err         error
	safeToRetry bool
}

func (e *pgconnError) Error() string {
	if e.msg == "" {
		return e.err.Error()
	}
	if e.err == nil {
		return e.msg
	}
	return fmt.Sprintf("%s: %s", e.msg, e.err.Error())
}

func (e *pgconnError) SafeToRetry() bool {
	return e.safeToRetry
}

func (e *pgconnError) Unwrap() error {
	return e.err
}

// errTimeout occurs when an error was caused by a timeout. Specifically, it wraps an error which is
// context.Canceled, context.DeadlineExceeded, or an implementer of net.Error where Timeout() is true.

// errTimout 发生在错误由超时引起的。具体来说，它包装了一个错误，该错误是 context.Canceled、context.DeadlineExceeded
// 或 net.Error 的实现，其中 Timeout() 为 true。
type errTimeout struct {
	err error
}

func (e *errTimeout) Error() string {
	return fmt.Sprintf("timeout: %s", e.err.Error())
}

func (e *errTimeout) SafeToRetry() bool {
	return SafeToRetry(e.err)
}

func (e *errTimeout) Unwrap() error {
	return e.err
}

type contextAlreadyDoneError struct {
	err error
}

func (e *contextAlreadyDoneError) Error() string {
	return fmt.Sprintf("context already done: %s", e.err.Error())
}

func (e *contextAlreadyDoneError) SafeToRetry() bool {
	return true
}

func (e *contextAlreadyDoneError) Unwrap() error {
	return e.err
}

// newContextAlreadyDoneError double-wraps a context error in `contextAlreadyDoneError` and `errTimeout`.
func newContextAlreadyDoneError(ctx context.Context) (err error) {
	return &errTimeout{&contextAlreadyDoneError{err: ctx.Err()}}
}

type writeError struct {
	err         error
	safeToRetry bool
}

func (e *writeError) Error() string {
	return fmt.Sprintf("write failed: %s", e.err.Error())
}

func (e *writeError) SafeToRetry() bool {
	return e.safeToRetry
}

func (e *writeError) Unwrap() error {
	return e.err
}

func redactPW(connString string) string {
	if strings.HasPrefix(connString, "postgres://") || strings.HasPrefix(connString, "postgresql://") {
		if u, err := url.Parse(connString); err == nil {
			return redactURL(u)
		}
	}
	quotedDSN := regexp.MustCompile(`password='[^']*'`)
	connString = quotedDSN.ReplaceAllLiteralString(connString, "password=xxxxx")
	plainDSN := regexp.MustCompile(`password=[^ ]*`)
	connString = plainDSN.ReplaceAllLiteralString(connString, "password=xxxxx")
	brokenURL := regexp.MustCompile(`:[^:@]+?@`)
	connString = brokenURL.ReplaceAllLiteralString(connString, ":xxxxxx@")
	return connString
}

func redactURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	if _, pwSet := u.User.Password(); pwSet {
		u.User = url.UserPassword(u.User.Username(), "xxxxx")
	}
	return u.String()
}

type NotPreferredError struct {
	err         error
	safeToRetry bool
}

func (e *NotPreferredError) Error() string {
	return fmt.Sprintf("standby server not found: %s", e.err.Error())
}

func (e *NotPreferredError) SafeToRetry() bool {
	return e.safeToRetry
}

func (e *NotPreferredError) Unwrap() error {
	return e.err
}
