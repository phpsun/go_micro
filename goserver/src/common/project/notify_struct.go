package project

// KYC结果通知消息
type MQKycResultNotify struct {
	UserId    int64                `json:"user_id"`
	KycResult *KycResultInfoNotify `json:"kyc_result"`
	// Fields example:
	//    "Dob": "1992-12-07",
	//    "Gender": "M",
	//    "idType": "id",
	//    "number": "142733199212075717",
	//    "validFrom": "2014-09-16",
	//    "validThrough": "2024-09-16",
	//    "overall": "ok"
}

// 支付系统通知消息
type MQPaymentNotify struct {
	TransId int64             `json:"trans_id"` // 订单ID
	Status  string            `json:"status"`   // 订单状态：0成功/1失败
	Channel string            `json:"channel"`  // 支付渠道：trustpayments/simplex
	Details map[string]string `json:"details"`  // 第三方返回结果详情
	// Fields example:
	//    "errorcode": "0",
	//    "notificationreference": "60-862963",
	//    "orderreference": "myorder12345",
	//    "paymenttypedescription": "VISA",
	//    "requestreference": "P60-mhdvrk3n",
	//    "responsesitesecurity": "7af96e9b60a46b12ba4b3c070586c30abe37c190cf9b7d2d42b2d69cddad6033",
	//    "settlestatus": "0",
	//    "sitereference": "test_opera81914",
	//    "transactionreference": "60-9-267724"
}

// 取消交易锁定额通知消息
type MQCancelAmountNotify struct {
	TransId int64  `json:"trans_id"`
	UserId  int64  `json:"user_id"`
	Amount  string `json:"amount"`
}

// 封号消息
type MQBanUserNotify struct {
	UserId int64  `json:"user_id"`
	Reason string `json:"reason"`
}

type KycResultInfoNotify struct {
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Dob           string `json:"dob"`
	Gender        string `json:"gender"`
	IdType        string `json:"id_type"`
	Number        string `json:"number"`
	ValidFrom     string `json:"valid_from"`
	ValidThrough  string `json:"valid_through"`
	Overall       string `json:"overall"`
	ImageDocFront string `json:"image_doc_front"`
	ImageDocBack  string `json:"image_doc_back"`
	ImageFace     string `json:"image_face"`
	Country       string `json:"country"`
}
