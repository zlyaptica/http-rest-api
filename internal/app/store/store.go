package store

// Store ...
type Store interface {
	User() UserRepository
	Post() PostRepository
	Star() StarRepository
}
