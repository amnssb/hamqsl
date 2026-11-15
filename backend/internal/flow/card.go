package flow

import "qsl-management/internal/model"

// 卡片进度以 flow_status 为唯一事实来源。布尔位是它的投影，只在这里改写。
//
//	PENDING_ISSUE --issue--> ISSUED --pack--> PACKED
//	                              \             \
//	                               +----send-----+--> SENT --receive--> RECEIVED
//	                                                    \                  \
//	                                                     +----confirm-------+--> SIGNED
//
//	任意进度 --resend--> PENDING_ISSUE
//
// 打包没有单独的 HTTP 接口（界面上也没有这个按钮），但 PACKED 仍是合法中间态，
// 发信同时接受「已制卡」和「已打包」。QSO / SWL / EYEBALL 的卡片一旦建立，
// 走的是同一条进度；场景差异在建卡入口和 SWL 反寄上，不在制卡/发信/签收上。
const (
	StatusPendingIssue = "PENDING_ISSUE"
	StatusIssued       = "ISSUED"
	StatusPacked       = "PACKED"
	StatusSent         = "SENT"
	StatusReceived     = "RECEIVED"
	StatusSigned       = "SIGNED"
	StatusError        = "ERROR"
)

// SendInput 发信时一并写入的邮寄信息。空的邮寄字段表示「沿用卡片上已有的值」，
// 备注则始终按本次请求覆盖（和原来的发信接口一致，传空串就是清空）。
type SendInput struct {
	At              string
	Remarks         string
	MailType        string
	TrackingNumber  string
	TrackingCarrier string
}

// ReceiveInput 管理端登记「对方已收卡」。
// UpdateRemarks 为假时不动备注，避免收卡记录关联把卡片上原有备注清掉。
type ReceiveInput struct {
	At            string
	Remarks       string
	UpdateRemarks bool
	RecordCode    string
}

func statusOf(card *model.CardRecord) string {
	if card.FlowStatus == "" {
		return StatusPendingIssue
	}
	return card.FlowStatus
}

// applyBools 按 flow_status 重写进度布尔位，并清掉已经不成立的步骤时间。
// 单号、邮件发送结果、备注不在这里动：单号可以在寄出前先填上，邮件状态只有重发才清。
func applyBools(card *model.CardRecord, status string) {
	switch status {
	case StatusPendingIssue:
		card.CardIssued = false
		card.EnvelopePrinted = false
		card.CardSent = false
		card.CardReceived = false
		card.ReceiptConfirmed = false
	case StatusIssued:
		card.CardIssued = true
		card.EnvelopePrinted = false
		card.CardSent = false
		card.CardReceived = false
		card.ReceiptConfirmed = false
	case StatusPacked:
		card.CardIssued = true
		card.EnvelopePrinted = true
		card.CardSent = false
		card.CardReceived = false
		card.ReceiptConfirmed = false
	case StatusSent:
		card.CardIssued = true
		card.CardSent = true
		card.CardReceived = false
		card.ReceiptConfirmed = false
	case StatusReceived:
		card.CardIssued = true
		card.CardSent = true
		card.CardReceived = true
		card.ReceiptConfirmed = false
	case StatusSigned:
		card.CardIssued = true
		card.CardSent = true
		card.CardReceived = true
		card.ReceiptConfirmed = true
	default:
		return
	}
	if !card.CardIssued {
		card.CardIssuedAt = ""
	}
	if !card.CardSent {
		card.SentAt = ""
	}
	if !card.CardReceived {
		card.ReceivedAt = ""
	}
}

func setStatus(card *model.CardRecord, status string) {
	applyBools(card, status)
	card.FlowStatus = status
}

// Align 用 flow_status 覆盖布尔位。PUT 只改备注之类的非流程字段时也要走一遍，
// 否则历史脏数据会一直和状态对不上。未知状态原样留下，避免把认不出的值抹掉。
// 返回值表示有没有改动，调用方据此决定要不要再写一次库。
func Align(card *model.CardRecord) bool {
	status := statusOf(card)
	switch status {
	case StatusPendingIssue, StatusIssued, StatusPacked, StatusSent, StatusReceived, StatusSigned:
	default:
		return false
	}
	beforeIssued := card.CardIssued
	beforeEnv := card.EnvelopePrinted
	beforeSent := card.CardSent
	beforeRecv := card.CardReceived
	beforeSigned := card.ReceiptConfirmed
	beforeIssuedAt := card.CardIssuedAt
	beforeSentAt := card.SentAt
	beforeRecvAt := card.ReceivedAt
	beforeStatus := card.FlowStatus
	setStatus(card, status)
	return beforeStatus != card.FlowStatus ||
		beforeIssued != card.CardIssued ||
		beforeEnv != card.EnvelopePrinted ||
		beforeSent != card.CardSent ||
		beforeRecv != card.CardReceived ||
		beforeSigned != card.ReceiptConfirmed ||
		beforeIssuedAt != card.CardIssuedAt ||
		beforeSentAt != card.SentAt ||
		beforeRecvAt != card.ReceivedAt
}

// Issue 待制卡 → 已制卡。
func Issue(card *model.CardRecord, at string) error {
	if statusOf(card) != StatusPendingIssue {
		return reject(CodeCardNotPending, "卡片不在待制卡状态，不能制卡")
	}
	setStatus(card, StatusIssued)
	card.CardIssuedAt = at
	return nil
}

// Pack 已制卡 → 已打包。目前没有对应的 HTTP 接口，规则先留在状态机里，
// 避免以后补按钮时又在 handler 里直接改布尔位。
func Pack(card *model.CardRecord) error {
	if statusOf(card) != StatusIssued {
		return reject(CodeCardNotIssued, "卡片尚未制卡，不能打包")
	}
	setStatus(card, StatusPacked)
	return nil
}

// Send 已制卡或已打包 → 已寄出。
func Send(card *model.CardRecord, in SendInput) error {
	st := statusOf(card)
	if st != StatusIssued && st != StatusPacked {
		return reject(CodeCardNotIssued, "卡片尚未制卡，不能发信")
	}
	setStatus(card, StatusSent)
	// 从已打包过来的卡片要保留「已打包」。setStatus(SENT) 不会去改这个位。
	if st == StatusPacked {
		card.EnvelopePrinted = true
	}
	card.SentAt = in.At
	card.SentRemarks = in.Remarks
	if in.MailType != "" {
		card.MailType = in.MailType
	}
	if in.TrackingNumber != "" {
		card.TrackingNumber = in.TrackingNumber
	}
	if in.TrackingCarrier != "" {
		card.TrackingCarrier = in.TrackingCarrier
	}
	return nil
}

// Receive 已寄出 → 对方已收（管理端登记，或收卡记录关联到这张发出的卡）。
// 这一步不代表对方在公开页签收，所以不会变成 SIGNED。
func Receive(card *model.CardRecord, in ReceiveInput) error {
	switch statusOf(card) {
	case StatusSent:
	case StatusReceived:
		return reject(CodeCardAlreadyRecv, "该卡片已登记收卡")
	case StatusSigned:
		return reject(CodeCardAlreadySigned, "该卡片已签收，不能再登记收卡")
	default:
		return reject(CodeCardNotSent, "卡片尚未寄出，不能登记收卡")
	}
	setStatus(card, StatusReceived)
	card.ReceivedAt = in.At
	if in.UpdateRemarks {
		card.ReceivedRemarks = in.Remarks
	}
	if in.RecordCode != "" {
		card.ReceivedRecordCode = in.RecordCode
	}
	return nil
}

// ConfirmReceipt 对方在公开页确认收件：已寄出或已登记收卡 → 已签收。
// 签收同时把「对方已收」写上，列表和总览不用再猜。
func ConfirmReceipt(card *model.CardRecord, at, remarks string) error {
	switch statusOf(card) {
	case StatusSent, StatusReceived:
	case StatusSigned:
		return reject(CodeCardAlreadySigned, "该卡片已确认收件")
	default:
		return reject(CodeCardNotSent, "卡片尚未寄出，暂不能确认收件")
	}
	setStatus(card, StatusSigned)
	card.ReceivedAt = at
	card.ReceivedRemarks = remarks
	return nil
}

// Resend 打回待制卡。已签收的布尔位必须一起清掉，否则下一次对齐会把状态算回已签收。
func Resend(card *model.CardRecord) error {
	setStatus(card, StatusPendingIssue)
	card.TrackingNumber = ""
	card.TrackingStatus = ""
	card.TrackingDetail = ""
	card.TrackingUpdatedAt = ""
	card.CreatedMailStatus = ""
	card.SentMailStatus = ""
	card.ReceivedMailStatus = ""
	card.ReceivedRecordCode = ""
	return nil
}

// SetReturnEnabled 后台按卡打开或关闭回寄引导。这是开关，不是进度，任何阶段都允许，
// 这样详情里的开关不会因为卡片已经寄出就按不下去。
func SetReturnEnabled(card *model.CardRecord, enabled bool) {
	card.ReturnMailEnabled = enabled
}

// RegisterCardReturn 对方确认收件后登记回寄方式。已签收之前、或没开回寄，都不能登记。
// 管理员尚未确认收到之前可以反复改单号（公开页有「更新回寄」）。
func RegisterCardReturn(card *model.CardRecord, mailType, tracking, at string) error {
	if statusOf(card) != StatusSigned {
		return reject(CodeReturnNeedSign, "请先确认收件，再登记回寄")
	}
	if !card.ReturnMailEnabled {
		return reject(CodeReturnDisabled, "本次寄件未开通回寄服务，请联系台站管理员")
	}
	card.ReturnMailType = mailType
	card.ReturnTracking = tracking
	card.ReturnMailedAt = at
	return nil
}

// PrepareCardReturn 管理员确认收到回寄之前的检查。文案和原来的接口保持一致。
func PrepareCardReturn(card *model.CardRecord) error {
	if card.ReturnMailedAt == "" {
		return reject(CodeReturnNotMailed, "对方尚未登记回寄，无法确认收卡")
	}
	if card.ReturnReceivedAt != "" {
		return reject(CodeReturnAlready, "该卡回寄已确认过收卡（记录 "+card.ReturnRecordCode+"）")
	}
	return nil
}
