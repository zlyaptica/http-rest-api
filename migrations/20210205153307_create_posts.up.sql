CREATE TABLE posts (
  id bigserial not null,
  author_id bigserial not null,
  header varchar not null,
  text_post varchar not null,
  PRIMARY KEY(id),
  CONSTRAINT fk_user
    FOREIGN KEY(author_id) 
	    REFERENCES users(id)
);