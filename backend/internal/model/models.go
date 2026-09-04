package model

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:50;uniqueIndex;not null" json:"username"`
	HashPassword string    `gorm:"size:128;not null" json:"-"`
	DisplayName  string    `gorm:"size:100" json:"display_name"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	IsAdmin      bool      `gorm:"default:false" json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type StationProfile struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CallSign    string    `gorm:"size:20;uniqueIndex;not null" json:"call_sign"`
	Name        string    `gorm:"size:100" json:"name"`
	NameEn      string    `gorm:"size:100" json:"name_en"`
	Telephone   string    `gorm:"size:30" json:"telephone"`
	PostalCode  string    `gorm:"size:10" json:"postal_code"`
	Address     string    `gorm:"type:text" json:"address"`
	AddressEn   string    `gorm:"type:text" json:"address_en"`
	Email       string    `gorm:"size:200" json:"email"`
	Remarks     string    `gorm:"type:text" json:"remarks"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type StationCard struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	CardVersion       string    `gorm:"size:100;uniqueIndex;not null" json:"card_version"`
	ImagePath         string    `gorm:"size:500" json:"image_path"`
	AvailableInventory int      `gorm:"default:0" json:"available_inventory"`
	VersionTotal      int       `gorm:"default:0" json:"version_total"`
	SortOrder         int       `gorm:"default:0" json:"sort_order"`
	QSOOnly           bool      `gorm:"default:false" json:"qso_only"`
	Remarks           string    `gorm:"type:text" json:"remarks"`
	IsActive          bool      `gorm:"default:true" json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type StationEquipment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RigName   string    `gorm:"size:100;not null" json:"rig_name"`
	Antennas  string    `gorm:"type:text;default:'[]'" json:"-"`
	Powers    string    `gorm:"type:text;default:'[]'" json:"-"`
	Modes     string    `gorm:"type:text;default:'[]'" json:"-"`
	Remarks   string    `gorm:"type:text" json:"remarks"`
	IsEnabled bool      `gorm:"default:true" json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type QsoRecord struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RecordCode string    `gorm:"size:30;uniqueIndex;not null" json:"record_code"`
	SceneType  string    `gorm:"size:20;default:QSO" json:"scene_type"`
	Date       string    `gorm:"size:10;not null" json:"date"`
	Time       string    `gorm:"size:10" json:"time"`
	Timezone   string    `gorm:"size:10;default:UTC+8" json:"timezone"`
	Freq       string    `gorm:"size:30" json:"freq"`
	Band       string    `gorm:"size:20" json:"band"`
	Mode       string    `gorm:"size:20" json:"mode"`
	MyRig      string    `gorm:"size:100" json:"my_rig"`
	MyRigMode  string    `gorm:"size:20" json:"my_rig_mode"`
	MyRigAnt   string    `gorm:"size:100" json:"my_rig_ant"`
	MyRigPwr   string    `gorm:"size:20" json:"my_rig_pwr"`
	MyQTH      string    `gorm:"size:200" json:"my_qth"`
	Operator   string    `gorm:"size:50" json:"operator"`
	CallSign   string    `gorm:"size:20;not null;index" json:"call_sign"`
	Rig        string    `gorm:"size:100" json:"rig"`
	Ant        string    `gorm:"size:100" json:"ant"`
	Pwr        string    `gorm:"size:20" json:"pwr"`
	QTH        string    `gorm:"size:200" json:"qth"`
	RstSent    string    `gorm:"size:10" json:"rst_sent"`
	RstRcvd    string    `gorm:"size:10" json:"rst_rcvd"`
	Remarks    string    `gorm:"type:text" json:"remarks"`
	HasCard    bool      `gorm:"default:false" json:"has_card"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CardRecord struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	CardCode           string    `gorm:"size:30;uniqueIndex;not null" json:"card_code"`
	CallSign           string    `gorm:"size:20;not null;index" json:"call_sign"`
	OwnerName          string    `gorm:"size:100" json:"owner_name"`
	CardType           string    `gorm:"size:30;default:QSO" json:"card_type"`
	SceneType          string    `gorm:"size:30;default:QSO" json:"scene_type"`
	CardVersion        string    `gorm:"size:100" json:"card_version"`
	QsoRecordID        *uint     `json:"qso_record_id"`
	OfflineActivityID  *uint     `json:"offline_activity_id"`
	AddressEntryID     *uint     `json:"address_entry_id"`
	BureauID           *uint     `json:"bureau_id"`
	CardDate           string    `gorm:"size:10" json:"card_date"`
	CardTime           string    `gorm:"size:10" json:"card_time"`
	BusinessRemarks    string    `gorm:"type:text" json:"business_remarks"`
	CardRemarks        string    `gorm:"type:text" json:"card_remarks"`
	SentRemarks        string    `gorm:"type:text" json:"sent_remarks"`
	ReceivedRemarks    string    `gorm:"type:text" json:"received_remarks"`
	MailType           string    `gorm:"size:20;default:REGISTERED" json:"mail_type"`
	TrackingNumber     string    `gorm:"size:50" json:"tracking_number"`
	TrackingCarrier    string    `gorm:"size:30;default:CHINA_POST" json:"tracking_carrier"`
	FlowStatus         string    `gorm:"size:30;default:PENDING_ISSUE" json:"flow_status"`
	CardIssued         bool      `gorm:"default:false" json:"card_issued"`
	CardIssuedAt       string    `gorm:"size:30" json:"card_issued_at"`
	EnvelopePrinted    bool      `gorm:"default:false" json:"envelope_printed"`
	CardSent           bool      `gorm:"default:false" json:"card_sent"`
	SentAt             string    `gorm:"size:30" json:"sent_at"`
	CardReceived       bool      `gorm:"default:false" json:"card_received"`
	ReceivedAt         string    `gorm:"size:30" json:"received_at"`
	ReceiptConfirmed   bool      `gorm:"default:false" json:"receipt_confirmed"`
	TrackingStatus     string    `gorm:"size:30" json:"tracking_status"`
	TrackingDetail     string    `gorm:"type:text" json:"tracking_detail"`
	TrackingUpdatedAt  string    `gorm:"size:30" json:"tracking_updated_at"`
	MailTargetEmail    string    `gorm:"size:200" json:"mail_target_email"`
	CreatedMailStatus  string    `gorm:"size:20" json:"created_mail_status"`
	SentMailStatus     string    `gorm:"size:20" json:"sent_mail_status"`
	ReceivedMailStatus string    `gorm:"size:20" json:"received_mail_status"`
	ReceivedRecordCode string    `gorm:"size:50" json:"received_record_code"`

	// 对方回寄信息：确认收件后，对方在确认页获取本台地址、登记回寄方式与单号
	// ReturnMailEnabled 由管理员在后台按卡开启；ReturnReceivedAt 为后台确认收到回寄的时间
	ReturnMailEnabled bool   `gorm:"default:false" json:"return_mail_enabled"`
	ReturnMailType    string `gorm:"size:20" json:"return_mail_type"`
	ReturnTracking    string `gorm:"size:50" json:"return_tracking"`
	ReturnMailedAt    string `gorm:"size:30" json:"return_mailed_at"`
	ReturnReceivedAt  string `gorm:"size:30" json:"return_received_at"`
	ReturnRecordCode  string `gorm:"size:50" json:"return_record_code"`

	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type AddressBookEntry struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	CallSign           string    `gorm:"size:20;not null;index" json:"call_sign"`
	Name               string    `gorm:"size:100" json:"name"`
	Telephone          string    `gorm:"size:30" json:"telephone"`
	PostalCode         string    `gorm:"size:10" json:"postal_code"`
	DestinationCountry string    `gorm:"size:50" json:"destination_country"`
	Address            string    `gorm:"type:text" json:"address"`
	Email              string    `gorm:"size:200" json:"email"`
	AddressRemarks     string    `gorm:"type:text" json:"address_remarks"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type BureauEntry struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	BureauName         string    `gorm:"size:100;not null;index" json:"bureau_name"`
	Telephone          string    `gorm:"size:30" json:"telephone"`
	PostalCode         string    `gorm:"size:10" json:"postal_code"`
	DestinationCountry string    `gorm:"size:50" json:"destination_country"`
	Address            string    `gorm:"type:text" json:"address"`
	AddressRemarks     string    `gorm:"type:text" json:"address_remarks"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ExchangeRequest struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	RequestCode    string    `gorm:"size:30;uniqueIndex;not null" json:"request_code"`
	SceneType      string    `gorm:"size:20" json:"scene_type"`
	CallSign       string    `gorm:"size:20;not null;index" json:"call_sign"`
	CardVersion    string    `gorm:"size:200" json:"card_version"`
	UseBureau      bool      `gorm:"default:false" json:"use_bureau"`
	BureauID       *uint     `json:"bureau_id"`
	BureauName     string    `gorm:"size:200" json:"bureau_name"`
	Email          string    `gorm:"size:200" json:"email"`
	Name           string    `gorm:"size:100" json:"name"`
	Telephone      string    `gorm:"size:30" json:"telephone"`
	PostalCode     string    `gorm:"size:10" json:"postal_code"`
	Address        string    `gorm:"type:text" json:"address"`
	Remarks            string    `gorm:"type:text" json:"remarks"`

	// 换卡理由（必填）
	ApplicationReason  string    `gorm:"type:text" json:"application_reason"`

	// QSO 场景证据（scene_type=QSO 时必填）
	QsoDate            string    `gorm:"size:10" json:"qso_date"`
	QsoTime            string    `gorm:"size:10" json:"qso_time"`
	QsoFreq            string    `gorm:"size:30" json:"qso_freq"`
	QsoBand            string    `gorm:"size:20" json:"qso_band"`
	QsoMode            string    `gorm:"size:20" json:"qso_mode"`
	QsoRstSent         string    `gorm:"size:10" json:"qso_rst_sent"`
	QsoRstRcvd         string    `gorm:"size:10" json:"qso_rst_rcvd"`

	// EYEBALL 场景证据（scene_type=EYEBALL 时必填）
	EyeballDate        string    `gorm:"size:10" json:"eyeball_date"`
	EyeballTime        string    `gorm:"size:10" json:"eyeball_time"`
	EyeballActivity    string    `gorm:"size:200" json:"eyeball_activity"`
	EyeballLocation    string    `gorm:"size:300" json:"eyeball_location"`

	// SWL 场景证据（scene_type=SWL 时必填）
	SwlDate            string    `gorm:"size:10" json:"swl_date"`
	SwlTime            string    `gorm:"size:8" json:"swl_time"`
	SwlFreq            string    `gorm:"size:30" json:"swl_freq"`
	SwlBand            string    `gorm:"size:20" json:"swl_band"`
	SwlMode            string    `gorm:"size:20" json:"swl_mode"`

	ReviewStatus       string    `gorm:"size:20;default:PENDING" json:"review_status"`
	ReviewReason   string    `gorm:"type:text" json:"review_reason"`
	ReviewedBy     string    `gorm:"size:50" json:"reviewed_by"`
	ReviewedAt     string    `gorm:"size:30" json:"reviewed_at"`
	CardCreated    bool      `gorm:"default:false" json:"card_created"`
	CardCreatedAt  string    `gorm:"size:30" json:"card_created_at"`
	CreatedCardID  *uint     `json:"created_card_id"`

	// SWL 反寄流程：管理员发送的回寄地址 + 对方寄出后登记的单号
	ReturnAddressText string `gorm:"type:text" json:"return_address_text"`
	AddressSentAt     string `gorm:"size:30" json:"address_sent_at"`
	ReturnMailType    string `gorm:"size:20" json:"return_mail_type"`
	ReturnTracking    string `gorm:"size:50" json:"return_tracking"`
	ReturnMailedAt    string `gorm:"size:30" json:"return_mailed_at"`
	ReturnReceivedAt  string `gorm:"size:30" json:"return_received_at"`
	// EYEBALL 细分：OFFLINE 线下补换 / ONLINE 网络EYE
	EyeballType string `gorm:"size:20;default:OFFLINE" json:"eyeball_type"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type OfflineActivity struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ActivityCode    string    `gorm:"size:30;uniqueIndex;not null" json:"activity_code"`
	ActivityName    string    `gorm:"size:200;not null" json:"activity_name"`
	ActivityLocation string   `gorm:"size:300" json:"activity_location"`
	ActivityDate    string    `gorm:"size:10" json:"activity_date"`
	ActivityTime    string    `gorm:"size:10" json:"activity_time"`
	CardRemarks     string    `gorm:"type:text" json:"card_remarks"`
	WorkflowStatus  string    `gorm:"size:20;default:ACTIVE" json:"workflow_status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ReceiveRecord struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ReceiveCode     string    `gorm:"size:50;uniqueIndex;not null" json:"receive_code"`
	CallSign        string    `gorm:"size:20;not null;index" json:"call_sign"`
	CardType        string    `gorm:"size:20;default:QSO" json:"card_type"`
	BusinessType    string    `gorm:"size:20;default:QSO" json:"business_type"`
	OfflineActivityID *uint   `json:"offline_activity_id"`
	ReceivedDate    string    `gorm:"size:10;not null" json:"received_date"`
	ReceivedAt      string    `gorm:"size:30" json:"received_at"`
	OutboundCardID  *uint     `json:"outbound_card_id"`
	MatchStatus     string    `gorm:"size:20;default:MATCHED" json:"match_status"`
	MatchReason     string    `gorm:"type:text" json:"match_reason"`
	Remarks         string    `gorm:"type:text" json:"remarks"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type SystemSetting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"size:100;uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	Desc      string    `gorm:"size:300" json:"description"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Action       string    `gorm:"size:50;not null" json:"action"`
	ResourceType string    `gorm:"size:50" json:"resource_type"`
	ResourceID   string    `gorm:"size:50" json:"resource_id"`
	Operator     string    `gorm:"size:50" json:"operator"`
	Detail       string    `gorm:"type:text" json:"detail"`
	IPAddress    string    `gorm:"size:50" json:"ip_address"`
	CreatedAt    time.Time `json:"created_at"`
}
