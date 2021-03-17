CREATE TABLE stars (
    id bigserial not null PRIMARY KEY,
    liker_id bigserial not null REFERENCES users,
    post_id bigserial not null REFERENCES posts
)