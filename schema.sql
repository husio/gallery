CREATE TABLE images (
    image_id      TEXT NOT NULL PRIMARY KEY,
    width         INTEGER NOT NULL,
    height        INTEGER NOT NULL,
    orientation   INTEGER NOT NULL,
    created       TIMESTAMP NOT NULL
);


CREATE TABLE tags (
    tag_id       TEXT NOT NULL PRIMARY KEY,
    image_id     TEXT NOT NULL REFERENCES images(image_id),
    name         TEXT NOT NULL,
    value        TEXT NOT NULL,
    created      TIMESTAMP NOT NULL
);
