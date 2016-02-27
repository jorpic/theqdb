package util

// Question describes raw question as it parsed form HTML
type Question struct {
	ID      uint64
	JSON    string
	Answers []*Answer
}

// Answer describes raw answer as it parsed form HTML
type Answer struct {
	ID     uint64
	JSON   string
	UserID uint64
}
