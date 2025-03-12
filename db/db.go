package db

type DB struct{}

func NewDB() *DB {
	return &DB{}
}

func (db *DB) Close() error {
	return nil
}
