package util

type Question struct {
	Id      int
	Json    string
	Answers []*Answer
}

type Answer struct {
	Id     uint64
	Json   string
	UserId uint64
}
