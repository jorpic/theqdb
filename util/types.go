package util

type Question struct {
	Id      uint64
	Json    string
	Answers []*Answer
}

type Answer struct {
	Id     uint64
	Json   string
	UserId uint64
}
