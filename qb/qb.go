// Package provides trivial SQL query builder
package qb

import "bytes"

func Q(selectFrom string) Query {
	return &query{
		selectFrom: selectFrom,
		chunks:     make([]string, 0, 32),
		args:       make([]interface{}, 0, 8),
	}
}

type Query interface {
	Where(string, ...interface{}) Query
	Limit(limit, offset int64) Query
	OrderBy(string) Query
	Build() (string, []interface{})
	String() string
}

type query struct {
	selectFrom string
	chunks     []string
	args       []interface{}
	order      string
	limit      int64
	offset     int64
}

func (q *query) OrderBy(o string) Query {
	q.order = o
	return q
}

func (q *query) Where(sql string, args ...interface{}) Query {
	q.chunks = append(q.chunks, sql)
	q.args = append(q.args, args...)
	return q
}

func (q *query) Limit(limit, offset int64) Query {
	q.limit = limit
	q.offset = offset
	return q
}

func (q *query) Build() (string, []interface{}) {
	args := q.args
	if q.limit != 0 {
		args = append(args, q.limit)
	}
	if q.offset != 0 {
		args = append(args, q.offset)
	}
	return q.String(), args
}

// TODO: optimize query building
func (q *query) String() string {
	var sql bytes.Buffer

	if q.selectFrom != "" {
		sql.WriteString(q.selectFrom)
	}

	if len(q.chunks) != 0 {
		if q.selectFrom != "" {
			sql.WriteByte(' ')
		}

		sql.WriteString("WHERE ")
		for i, c := range q.chunks {
			if i != 0 {
				sql.WriteString(" AND ")
			}
			sql.WriteByte('(')
			sql.WriteString(c)
			sql.WriteByte(')')
		}
	}

	if q.order != "" {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(q.order)
	}

	if q.limit != 0 {
		sql.WriteString(" LIMIT ?")
	}
	if q.offset != 0 {
		sql.WriteString(" OFFSET ?")
	}

	return sql.String()
}
