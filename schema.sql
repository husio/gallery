CREATE TABLE images (
    image_id      TEXT NOT NULL PRIMARY KEY,
    width         INTEGER NOT NULL,
    height        INTEGER NOT NULL,
    orientation   INTEGER NOT NULL,
    created       TIMESTAMP NOT NULL
);


CREATE TABLE tags (
    name         TEXT NOT NULL,
    image_id     TEXT NOT NULL REFERENCES images(image_id),
    created      TIMESTAMP NOT NULL,

    PRIMARY KEY(name, image_id)
);
