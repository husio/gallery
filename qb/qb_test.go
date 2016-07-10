package qb

import "testing"

func TestWhere(t *testing.T) {
	type call struct {
		sql  string
		args []interface{}
	}

	cases := map[string]struct {
		q    Query
		want string
	}{
		"empty": {
			q:    Q(""),
			want: "",
		},
		"simple_select": {
			q: Q("SELECT a.foo, b.bar FROM a, b").Where(
				"a.b_id = b.b_id").Where("b.size = ?", 32),
			want: "SELECT a.foo, b.bar FROM a, b WHERE (a.b_id = b.b_id) AND (b.size = ?)",
		},
		"and_one": {
			q:    Q("").Where("color = ?", "red"),
			want: "WHERE (color = ?)",
		},
		"and_two": {
			q:    Q("").Where("color = ?", "red").Where("age = ?", 42),
			want: "WHERE (color = ?) AND (age = ?)",
		},
		"limit": {
			q:    Q("").Where("color = ?", "red").Limit(18, 0),
			want: "WHERE (color = ?) LIMIT ?",
		},
		"offset": {
			q:    Q("").Where("color = ?", "red").Limit(0, 82),
			want: "WHERE (color = ?) OFFSET ?",
		},
		"limit_offset": {
			q:    Q("").Where("color = ?", "red").Limit(18, 52),
			want: "WHERE (color = ?) LIMIT ? OFFSET ?",
		},
		"order_by": {
			q:    Q("").Where("color = ?", "blue").OrderBy("color DESC"),
			want: "WHERE (color = ?) ORDER BY color DESC",
		},
		"all": {
			q:    Q("SELECT * FROM foo").Where("color = ?", "blue").OrderBy("color").Limit(2, 3),
			want: "SELECT * FROM foo WHERE (color = ?) ORDER BY color LIMIT ? OFFSET ?",
		},
	}

	for tname, tc := range cases {
		if got := tc.q.String(); got != tc.want {
			t.Errorf("%s: \nwant: %s \n got: %s", tname, tc.want, got)
			continue
		}
	}
}
