package flow

import (
	"strings"

	"qsl-management/internal/model"
)

// 换卡申请的审核和 SWL 反寄。场景决定哪些边合法：
//
//	QSO / EYEBALL：PENDING --approve--> APPROVED --create card--> 卡片 PENDING_ISSUE
//	               PENDING --reject--> REJECTED
//
//	SWL：PENDING --approve--> APPROVED --send address--> 对方登记寄出 --admin receive--> 建卡
//	     没收到对方的卡之前，不能建回寄卡片。
const (
	SceneQSO     = "QSO"
	SceneSWL     = "SWL"
	SceneEyeball = "EYEBALL"

	ReviewPending  = "PENDING"
	ReviewApproved = "APPROVED"
	ReviewRejected = "REJECTED"
)

// NormScene 去掉空白并转成大写。空串保持空串，调用方自己决定要不要默认成 QSO。
func NormScene(scene string) string {
	return strings.ToUpper(strings.TrimSpace(scene))
}

// InitQSO 新建通联还没有卡。has_card 只能由建卡/删卡改写，这里强制写成假，
// 避免请求体里夹带 has_card=true。
func InitQSO(q *model.QsoRecord) {
	q.HasCard = false
	q.CallSign = strings.ToUpper(strings.TrimSpace(q.CallSign))
}

// InitExchange 新申请一律从待审核开始，建卡和反寄进度全部清空。
func InitExchange(ex *model.ExchangeRequest) {
	ex.ID = 0
	ex.CallSign = strings.ToUpper(strings.TrimSpace(ex.CallSign))
	ex.SceneType = NormScene(ex.SceneType)
	if ex.SceneType == "" {
		ex.SceneType = SceneQSO
	}
	ex.ReviewStatus = ReviewPending
	ex.ReviewReason = ""
	ex.ReviewedBy = ""
	ex.ReviewedAt = ""
	ex.CardCreated = false
	ex.CardCreatedAt = ""
	ex.CreatedCardID = nil
	ex.ReturnAddressText = ""
	ex.AddressSentAt = ""
	ex.ReturnMailType = ""
	ex.ReturnTracking = ""
	ex.ReturnMailedAt = ""
	ex.ReturnReceivedAt = ""
}

// Approve 待审核 → 已通过。
func Approve(ex *model.ExchangeRequest, by, at string) error {
	if ex.ReviewStatus != ReviewPending {
		return reject(CodeExNotPending, "该申请已处理")
	}
	ex.ReviewStatus = ReviewApproved
	ex.ReviewedBy = by
	ex.ReviewedAt = at
	return nil
}

// Reject 只允许从待审核拒绝。已通过的申请可能已经建卡或进入反寄，再拒绝会把两边状态拆开。
func Reject(ex *model.ExchangeRequest, reason, at string) error {
	if ex.ReviewStatus == ReviewRejected {
		return reject(CodeExAlreadyRejected, "该申请已是拒绝状态")
	}
	if ex.ReviewStatus != ReviewPending {
		return reject(CodeExNotPending, "该申请已处理")
	}
	ex.ReviewStatus = ReviewRejected
	ex.ReviewReason = reason
	ex.ReviewedAt = at
	return nil
}

// SendAddress SWL 已通过、且对方还没登记寄出时，写入（或覆盖）回寄地址。
func SendAddress(ex *model.ExchangeRequest, text, at string) error {
	if NormScene(ex.SceneType) != SceneSWL {
		return reject(CodeNotSWL, "仅 SWL 申请可以发送回寄地址")
	}
	if ex.ReviewStatus != ReviewApproved {
		return reject(CodeExNotApproved, "申请尚未通过，不能发送回寄地址")
	}
	if strings.TrimSpace(ex.ReturnMailedAt) != "" {
		return reject(CodeSWLAddressClosed, "对方已登记寄出，不能再改回寄地址")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return reject(CodeSWLAddressEmpty, "回寄地址不能为空")
	}
	ex.ReturnAddressText = text
	ex.AddressSentAt = at
	return nil
}

// MarkCounterpartMailed 对方在进度页登记寄出。仅 SWL、且已通过。收卡登记之后不能再改。
func MarkCounterpartMailed(ex *model.ExchangeRequest, mailType, tracking, at string) error {
	if NormScene(ex.SceneType) != SceneSWL {
		return reject(CodeNotSWL, "仅 SWL 申请需要登记寄出")
	}
	if ex.ReviewStatus != ReviewApproved {
		return reject(CodeExNotApproved, "申请尚未通过，暂不能登记寄出")
	}
	if strings.TrimSpace(ex.ReturnReceivedAt) != "" {
		return reject(CodeSWLAlreadyRecv, "该申请已登记过收卡")
	}
	ex.ReturnMailType = mailType
	ex.ReturnTracking = tracking
	ex.ReturnMailedAt = at
	return nil
}

// PrepareSWLReceive 管理员登记「我方已收到对方寄来的卡」之前的检查。
func PrepareSWLReceive(ex *model.ExchangeRequest) error {
	if NormScene(ex.SceneType) != SceneSWL {
		return reject(CodeNotSWL, "仅 SWL 申请可以登记收卡")
	}
	if ex.ReviewStatus != ReviewApproved {
		return reject(CodeExNotApproved, "申请尚未通过，暂不能登记收卡")
	}
	if strings.TrimSpace(ex.ReturnMailType) == "" {
		return reject(CodeSWLNotMailed, "对方尚未登记寄出信息")
	}
	if strings.TrimSpace(ex.ReturnReceivedAt) != "" {
		return reject(CodeSWLAlreadyRecv, "该申请已登记过收卡")
	}
	return nil
}

func canCreateFromExchange(ex *model.ExchangeRequest) error {
	if ex.ReviewStatus != ReviewApproved {
		return reject(CodeExNotApproved, "只有已通过的申请才能创建卡片")
	}
	if ex.CardCreated {
		return reject(CodeExCardExists, "该申请已创建卡片")
	}
	if NormScene(ex.SceneType) == SceneSWL && strings.TrimSpace(ex.ReturnReceivedAt) == "" {
		return reject(CodeExSWLWaiting, "SWL 申请需先收到对方寄来的卡片，才能创建回寄卡片")
	}
	return nil
}
