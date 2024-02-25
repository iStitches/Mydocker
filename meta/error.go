package meta

import (
	"fmt"
	"strings"
)

// used to shift category on error code
const CategoryBitOnErrorCode = 24

// ErrCode is the error code of Mydocker
// Usually the left 8bits represent category of error code and right 24bits represent the behavior of error
type ErrCode uint32

const (
	// unsupport type error
	ErrUnsupportedType ErrCode = 1 + iota
	// exceed stack limit error
	ErrStackOverflow
	// read error
	ErrRead
	// write error
	ErrWrite
	// disMatch type error
	ErrDismatchType
	// convert error
	ErrConvert
	// not found error
	ErrNotFound
	// invalid parameter error
	ErrInvalidParam
	// mount failed
	ErrMount
	// unMount failed
	ErrUnMount
	// ip link failed
	ErrLink
	// ip addr failed
	ErrAddr
	// iptables failed
	ErrIptables
	// network-driver exec failed
	ErrDriverExec
	// network-ipam exec failed
	ErrIpamExec
)

var errMap = map[ErrCode]string{
	ErrUnsupportedType: "unsupportted type",
	ErrStackOverflow:   "exceed depth limit",
	ErrRead:            "read failed",
	ErrWrite:           "write failed",
	ErrDismatchType:    "dismatchedType",
	ErrConvert:         "convert failed",
	ErrNotFound:        "not found",
	ErrInvalidParam:    "invalid parameter",
	ErrMount:           "mount failed",
	ErrUnMount:         "unmount_failed",
	ErrLink:            "ip link failed",
	ErrAddr:            "ip addr failed",
	ErrIptables:        "iptables exec failed",
	ErrDriverExec:      "network driver exec failed",
	ErrIpamExec:        "network ipam exec failed",
}

func NewErrorCode(behavior ErrCode, category Category) ErrCode {
	return behavior | (ErrCode(category) << CategoryBitOnErrorCode)
}

func (ec ErrCode) Category() Category {
	return Category(ec >> CategoryBitOnErrorCode)
}

func (ec ErrCode) Behavior() ErrCode {
	return ErrCode(ec & 0x00ffffff)
}

func (ec ErrCode) Error() string {
	return ec.String()
}

func (ec ErrCode) String() string {
	if m, ok := errMap[ec.Behavior()]; ok {
		return m
	} else {
		return fmt.Sprintf("error code %d", ec)
	}
}

// Error is the error concrete type of Mydocker
type Error struct {
	Code ErrCode
	Msg  string
	Err  error
}

// NewError creates a new error with error code, message and preceding error
func NewError(code ErrCode, msg string, err error) error {
	return Error{
		Code: code,
		Msg:  msg,
		Err:  err,
	}
}

// Message return current message if has,
// otherwise return preceding error message
func (err Error) Message() string {
	output := []string{err.Msg}
	if err.Err != nil {
		output = append(output, err.Err.Error())
	}
	return strings.Join(output, "\n")
}

// Error return error message,
// combining category, behavior and message
func (err Error) Error() string {
	return fmt.Sprintf("[%s] %s: %s", err.Code.Category(), err.Code.Behavior(), err)
}

func (err Error) Unwrap() error {
	return err.Err
}
