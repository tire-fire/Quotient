package checks

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedRowDriver is an up SQL server that answers every query with one row
// whose first column is "count(*)".
type fixedRowDriver struct{}
type fixedRowConn struct{}
type fixedRowRows struct{ sent bool }

func (fixedRowDriver) Open(string) (driver.Conn, error) { return fixedRowConn{}, nil }
func (fixedRowConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not supported")
}
func (fixedRowConn) Close() error               { return nil }
func (fixedRowConn) Begin() (driver.Tx, error)  { return nil, errors.New("not supported") }
func (fixedRowConn) Ping(context.Context) error { return nil }
func (fixedRowConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &fixedRowRows{}, nil
}
func (*fixedRowRows) Columns() []string { return []string{"c"} }
func (*fixedRowRows) Close() error      { return nil }
func (r *fixedRowRows) Next(dest []driver.Value) error {
	if r.sent {
		return io.EOF
	}
	r.sent = true
	dest[0] = []byte("count(*)")
	return nil
}

func init() { sql.Register("fixedrow", fixedRowDriver{}) }

// Verify compiles Output only with UseRegex, so Run must compare an exact-match
// Output literally, not MustCompile it.
func TestSqlExactOutputWithRegexMetacharacters(t *testing.T) {
	c := &Sql{
		Service: Service{Name: "box01-sql", Display: "sql", Target: "192.0.2.1", Port: 3306, Timeout: 5, Attempts: 1,
			TaskCredentials: []TaskCredential{{Username: "u", Password: "p"}}},
		Kind:  "fixedrow",
		Query: []queryData{{Command: "SELECT count(*) FROM t", Output: "count(*)"}},
	}
	require.NoError(t, c.Verify("box01", "192.0.2.1", 10, 5, 0, 0))

	for _, tc := range []struct {
		name   string
		regex  bool
		output string
		want   bool
	}{
		{"literal metacharacters", false, "count(*)", true},
		{"pattern is not a literal match", false, `^count`, false},
		{"regex matches prefix", true, `^count`, true},
		{"regex misses", true, `^sum`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c.Query[0].UseRegex = tc.regex
			c.Query[0].Output = tc.output
			results := make(chan Result, 1)
			c.Run(1, "01", 1, results)
			r := <-results
			assert.Equal(t, tc.want, r.Status, "error=%q debug=%q", r.Error, r.Debug)
		})
	}
}
