package storage

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/husio/gallery/qb"
	"github.com/husio/gallery/sq"
)

type Image struct {
	ImageID     string    `db:"image_id"    json:"imageId"`
	Width       int       `db:"width"       json:"width"`
	Height      int       `db:"height"      json:"height"`
	Orientation int       `db:"orientation" json:"orientation"`
	Created     time.Time `db:"created"     json:"created"`
	Tags        []*Tag    `db:"-"           json:"tags"`
}

type Tag struct {
	TagID   string    `db:"tag_id"   json:"tagId"`
	ImageID string    `db:"image_id" json:"imageId"`
	Name    string    `db:"name"     json:"name"`
	Value   string    `db:"value"    json:"value"`
	Created time.Time `db:"created"  json:"created"`
}

func CreateTag(e sq.Execer, tag Tag) (*Tag, error) {
	oid := sha256.New()
	fmt.Fprint(oid, tag.ImageID)
	fmt.Fprint(oid, tag.Name)
	fmt.Fprint(oid, tag.Value)
	tag.TagID = encode(oid)

	if tag.Created.IsZero() {
		tag.Created = time.Now()
	}

	_, err := e.Exec(`
		INSERT INTO tags (tag_id, image_id, name, value, created)
		VALUES (?, ?, ?, ?, ?)
	`, tag.TagID, tag.ImageID, tag.Name, tag.Value, tag.Created)
	return &tag, sq.CastErr(err)
}

func Images(s sq.Selector, opts ImagesOpts) ([]*Image, error) {
	var q qb.Query
	if len(opts.Tags) == 0 {
		q = qb.Q("SELECT i.* FROM images i")
	} else {
		q = qb.Q("SELECT i.* FROM images i INNER JOIN tags t ON i.image_id = t.image_id")
		for _, kv := range opts.Tags {
			q.Where("t.name = ? AND t.value = ?", kv.Key, kv.Value)
		}
	}

	q.OrderBy("i.created DESC").Limit(opts.Limit, opts.Offset)
	query, args := q.Build()

	var imgs []*Image
	err := s.Select(&imgs, query, args...)
	return imgs, sq.CastErr(err)
}

type ImagesOpts struct {
	Limit  int64
	Offset int64
	Tags   []KeyValue
}

type KeyValue struct {
	Key   string
	Value string
}

func CreateImage(e sq.Execer, img Image) (*Image, error) {
	_, err := e.Exec(`
		INSERT INTO images (image_id, width, height, created, orientation)
		VALUES (?, ?, ?, ?, ?)
	`, img.ImageID, img.Width, img.Height, img.Created, img.Orientation)
	return &img, sq.CastErr(err)
}

func ImageByID(g sq.Getter, imageID string) (*Image, error) {
	var img Image
	err := g.Get(&img, `
		SELECT * FROM images
		WHERE image_id = ?
		LIMIT 1
	`, imageID)
	if err != nil {
		return nil, sq.CastErr(err)
	}
	return &img, nil
}

func ImageTags(s sq.Selector, imageID string) ([]*Tag, error) {
	var tags []*Tag
	err := s.Select(&tags, `
		SELECT * FROM tags
		WHERE image_id = ?
		LIMIT 1000
	`, imageID)
	if err != nil {
		return nil, sq.CastErr(err)
	}
	return tags, nil
}

func TagGroups(s sq.Selector) ([]*TagGroup, error) {
	var tags []*TagGroup
	err := s.Select(&tags, `
                SELECT name, value, sum(1) AS count
                FROM tags
                GROUP BY name, value
        `)
	return tags, sq.CastErr(err)
}

type TagGroup struct {
	Name  string `db:"name"     json:"name"`
	Value string `db:"value"    json:"value"`
	Count int    `db:"count"    json:"count"`
}

func encode(h hasher) string {
	s := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return strings.TrimRight(s, "=")
}

type hasher interface {
	Sum(b []byte) []byte
}
