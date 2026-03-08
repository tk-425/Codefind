package query

type Options struct {
	Text     string
	RepoID   string
	Project  string
	Language string
	All      bool
	Page     int
	PageSize int
	TopK     int
}
