package flow

import "errors"

// 非法迁移一律 HTTP 422。code 沿用原来的 QSL-422 前缀，后面加上稳定的原因，
// 调用方可以按 code 分支；已经在用的文案保持原句，方便前端直接展示。
const (
	CodeCardNotPending    = "QSL-422-CARD-NOT-PENDING"
	CodeCardNotIssued     = "QSL-422-CARD-NOT-ISSUED"
	CodeCardNotSent       = "QSL-422-CARD-NOT-SENT"
	CodeCardAlreadyRecv   = "QSL-422-CARD-ALREADY-RECEIVED"
	CodeCardAlreadySigned = "QSL-422-CARD-ALREADY-SIGNED"
	CodeQsoHasCard        = "QSL-422-QSO-HAS-CARD"
	CodeQsoScene          = "QSL-422-QSO-SCENE"
	CodeExNotPending      = "QSL-422-EX-NOT-PENDING"
	CodeExAlreadyRejected = "QSL-422-EX-ALREADY-REJECTED"
	CodeExNotApproved     = "QSL-422-EX-NOT-APPROVED"
	CodeExCardExists      = "QSL-422-EX-CARD-EXISTS"
	CodeExSWLWaiting      = "QSL-422-EX-SWL-WAITING"
	CodeNotSWL            = "QSL-422-NOT-SWL"
	CodeSWLNotMailed      = "QSL-422-SWL-NOT-MAILED"
	CodeSWLAlreadyRecv    = "QSL-422-SWL-ALREADY-RECEIVED"
	CodeSWLAddressClosed  = "QSL-422-SWL-ADDRESS-CLOSED"
	CodeSWLAddressEmpty   = "QSL-422-SWL-ADDRESS-EMPTY"
	CodeReturnNeedSign    = "QSL-422-RETURN-NEED-SIGN"
	CodeReturnDisabled    = "QSL-422-RETURN-DISABLED"
	CodeReturnNotMailed   = "QSL-422-RETURN-NOT-MAILED"
	CodeReturnAlready     = "QSL-422-RETURN-ALREADY-RECEIVED"
)

// Error 是一次被拒绝的迁移。
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func reject(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// AsError 取出迁移错误。普通数据库错误不是迁移错误。
func AsError(err error) (*Error, bool) {
	var fe *Error
	if errors.As(err, &fe) {
		return fe, true
	}
	return nil, false
}
