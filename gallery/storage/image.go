package storage

import (
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
	Name    string    `db:"name"   json:"name"`
	ImageID string    `db:"image_id" json:"imageId"`
	Created time.Time `db:"created"  json:"created"`
}

func CreateTag(e sq.Execer, tag Tag) (*Tag, error) {
	if tag.Created.IsZero() {
		tag.Created = time.Now()
	}
	_, err := e.Exec(`
		INSERT INTO tags (image_id, name, created)
		VALUES (?, ?, ?)
	`, tag.ImageID, tag.Name, tag.Created)
	return &tag, sq.CastErr(err)
}

func Images(s sq.Selector, opts ImagesOpts) ([]*Image, error) {
	var q qb.Query
	if len(opts.Tags) == 0 {
		q = qb.Q("SELECT i.* FROM images i")
	} else {
		q = qb.Q("SELECT i.* FROM images i INNER JOIN tags t ON i.image_id = t.image_id")
		for _, name := range opts.Tags {
			q.Where("t.name = ?", name)
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
	Tags   []string
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
                SELECT name, sum(1) AS count
                FROM tags
                GROUP BY name
        `)
	return tags, sq.CastErr(err)
}

type TagGroup struct {
	Name  string `db:"name"     json:"name"`
	Count int    `db:"count"    json:"count"`
}
