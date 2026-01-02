package db

type SQLClient interface {
	Select() error
}
