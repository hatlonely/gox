package rdb

type SQLOptions struct {
	Host string `cfg:"host" def:"localhost"`
	Port string `cfg:"port" def:"3306"`
}

type SQL struct {
}

func NewSQLWithOptions(options *SQLOptions) (*SQL, error) {
	return &SQL{}, nil
}

type SQLRecord struct{}

type SQLRecordBuilder struct{}
