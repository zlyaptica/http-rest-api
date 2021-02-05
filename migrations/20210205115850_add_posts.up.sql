CREATE TABLE posts (
    id bigserial not null primary key,
    author_id bigserial not null foreign key,
    header varchar not null,
    text_post varchar not null
)