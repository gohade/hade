package xerror

type Code int32

const (
	OK Code = iota
	Canceled
	Unknown
	InvalidArgument
	NotFound
	MysqlErr
	RedisErr
	UnknownCode = -9999
)

// msgs ...
var msgs = map[Code]*err{
	OK:              New(OK, "success"),
	Canceled:        New(Canceled, "canceled"),
	Unknown:         New(Unknown, "unknown"),
	InvalidArgument: New(InvalidArgument, "invalid argument"),
	NotFound:        New(NotFound, "not found"),
	MysqlErr:        New(MysqlErr, "mysql err"),
	RedisErr:        New(RedisErr, "redis Err"),
	UnknownCode:     New(UnknownCode, "unknown code"),
}

// GetErr ...
func GetErr(code Code) *err {
	if val, ok := msgs[code]; ok {
		return val
	}

	return msgs[UnknownCode]
}
