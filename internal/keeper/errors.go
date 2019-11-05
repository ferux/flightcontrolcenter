package keeper

type TempError string

func (err TempError) Error() string {
	return "keeper temp error: " + string(err)
}

func (err TempError) IsTemporary() bool { return true }

type TemporaryError interface {
	IsTemporary() bool
}

// IsTemporary checks if the error will gone later.
func IsTemporary(err error) bool {
	_, ok := err.(TemporaryError)
	return ok
}

type PermamentError string

func (err PermamentError) Error() string {
	return "keeper error: " + string(err)
}

const (
	ErrUnsupportedVersion PermamentError = "unsupported version"
	ErrUnexpectedPacket   PermamentError = "unexpected packet"
)

type NotFoundError string

func (err NotFoundError) Error() string {
	return "not found: " + string(err)
}
