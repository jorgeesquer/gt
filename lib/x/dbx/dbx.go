package dbx

import (
	"database/sql"
	"fmt"

	"github.com/gtlang/gt/lib/x/goql"
)

type Reader struct {
	columns []*Column
	rows    *sql.Rows
	values  []interface{}
}

func (r *Reader) Columns() ([]*Column, error) {
	if r.columns == nil {
		cols, err := getColumns(r.rows)
		if err != nil {
			return nil, err
		}
		r.columns = cols
	}
	return r.columns, nil
}

func (r *Reader) Next() bool {
	return r.rows.Next()
}

func (r *Reader) Read() ([]interface{}, error) {
	if r.values == nil {
		cols, err := r.rows.Columns()
		if err != nil {
			return nil, err
		}
		r.values = make([]interface{}, len(cols))
	}

	for i := range r.values {
		r.values[i] = &r.values[i]
	}

	if err := r.rows.Scan(r.values...); err != nil {
		return nil, err
	}

	cols, err := r.Columns()
	if err != nil {
		return nil, err
	}

	for i, v := range r.values {
		val, err := Convert(v, cols[i].Type)
		if err != nil {
			return nil, fmt.Errorf("error converting %s: %v", cols[i].Name, err)
		}
		r.values[i] = val
	}

	return r.values, nil
}

func (r *Reader) Err() error {
	return r.rows.Err()
}

func (r *Reader) Close() error {
	return r.rows.Close()
}

type Scanner interface {
	Scan(dest ...interface{}) error
}

func (db *DB) ShowQuery(query string) (*Table, error) {
	q, err := goql.ParseQuery(query)
	if err != nil {
		return nil, err
	}

	sq, ok := q.(*goql.ShowQuery)
	if !ok {
		return nil, fmt.Errorf("not a show query")
	}

	return db.ShowQueryEx(sq)
}

func (db *DB) ShowQueryEx(query *goql.ShowQuery) (*Table, error) {
	s, _, err := db.ToSql(query, nil)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryRaw(s)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return ToTable(rows)
}

func (db *DB) ShowReader(query string) (*Reader, error) {
	q, err := goql.ParseQuery(query)
	if err != nil {
		return nil, err
	}

	sq, ok := q.(*goql.ShowQuery)
	if !ok {
		return nil, fmt.Errorf("not a show query")
	}

	return db.ShowReaderEx(sq)
}

func (db *DB) ShowReaderEx(query *goql.ShowQuery) (*Reader, error) {
	s, _, err := db.ToSql(query, nil)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryRaw(s)
	if err != nil {
		return nil, err
	}

	return &Reader{rows: rows}, nil
}

func (db *DB) ReaderEx(query *goql.SelectQuery) (*Reader, error) {
	rows, err := db.QueryRowsEx(query)
	if err != nil {
		return nil, err
	}

	return &Reader{rows: rows}, nil
}

func (db *DB) Reader(query string, args ...interface{}) (*Reader, error) {
	rows, err := db.QueryRows(query, args...)
	if err != nil {
		return nil, err
	}

	return &Reader{rows: rows}, nil
}

func (db *DB) ReaderRaw(query string, args ...interface{}) (*Reader, error) {
	rows, err := db.QueryRaw(query, args...)
	if err != nil {
		return nil, err
	}

	return &Reader{rows: rows}, nil
}

func (db *DB) QueryEx(query *goql.SelectQuery) (*Table, error) {
	rows, err := db.QueryRowsEx(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return ToTable(rows)
}

func (db *DB) QueryRowsEx(q *goql.SelectQuery) (*sql.Rows, error) {
	s, params, err := db.ToSql(q, q.Params)
	if err != nil {
		return nil, err
	}

	return db.QueryRaw(s, params...)
}

func (db *DB) Query(query string, args ...interface{}) (*Table, error) {
	rows, err := db.QueryRows(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return ToTable(rows)
}

func (db *DB) QueryRows(query string, args ...interface{}) (*sql.Rows, error) {
	q, err := goql.ParseQuery(query)
	if err != nil {
		return nil, err
	}

	sq, ok := q.(*goql.SelectQuery)
	if !ok {
		return nil, fmt.Errorf("not a select query")
	}

	s, params, err := db.ToSql(sq, args)
	if err != nil {
		return nil, err
	}

	return db.QueryRaw(s, params...)
}

func (db *DB) QueryRow(query string, args ...interface{}) (*Row, error) {
	t, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	switch len(t.Rows) {
	case 0:
		return nil, nil
	case 1:
		return t.Rows[0], nil
	default:
		return nil, fmt.Errorf("the query returned %d results", len(t.Rows))
	}
}

func (db *DB) QueryRowEx(query *goql.SelectQuery) (*Row, error) {
	t, err := db.QueryEx(query)
	if err != nil {
		return nil, err
	}

	switch len(t.Rows) {
	case 0:
		return nil, nil
	case 1:
		return t.Rows[0], nil
	default:
		return nil, fmt.Errorf("the query returned %d results", len(t.Rows))
	}
}

func (db *DB) QueryValue(query string, args ...interface{}) (interface{}, error) {
	r, err := db.QueryRow(query, args...)
	if err != nil {
		return nil, err
	}

	if r == nil {
		return nil, nil
	}

	if len(r.Values) != 1 {
		return nil, fmt.Errorf("the query returned %d values", len(r.Values))
	}

	return r.Values[0], nil
}

func (db *DB) QueryValueRaw(query string, args ...interface{}) (interface{}, error) {
	rows, err := db.QueryRaw(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	t, more, err := ToTableLimit(rows, 1)
	if err != nil {
		return nil, err
	}
	if more {
		return nil, fmt.Errorf("the query returned more than one row")
	}

	if len(t.Rows) == 0 {
		return nil, nil
	}

	r := t.Rows[0]

	if len(r.Values) != 1 {
		return nil, fmt.Errorf("the query returned %d values", len(r.Values))
	}

	return r.Values[0], nil
}

func (db *DB) QueryValueEx(query *goql.SelectQuery) (interface{}, error) {
	r, err := db.QueryRowEx(query)
	if err != nil {
		return nil, err
	}

	if r == nil {
		return nil, nil
	}

	if len(r.Values) != 1 {
		return nil, fmt.Errorf("the query returned %d values", len(r.Values))
	}

	return r.Values[0], nil
}

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	if db.ReadOnly {
		return nil, ErrReadOnly
	}

	q, err := goql.ParseQuery(query)
	if err != nil {
		return nil, err
	}

	return db.ExecEx(q, args...)
}

func (db *DB) ExecEx(q goql.Query, args ...interface{}) (sql.Result, error) {
	if db.ReadOnly {
		return nil, ErrReadOnly
	}

	s, params, err := db.ToSql(q, args)
	if err != nil {
		return nil, err
	}

	r, err := db.ExecRaw(s, params...)
	if err != nil {
		return nil, err
	}

	return r, nil
}
