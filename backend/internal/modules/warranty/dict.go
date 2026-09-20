package warranty

var componentLabels = map[string]string{
	ComponentLamp: "灯具",
	ComponentPole: "灯杆",
}

var partyLabels = map[string]string{
	PartyManufacturer: "厂家",
	PartyInternal:     "自有班组",
}

var claimStatusLabels = map[string]string{
	ClaimStatusPending:    "待厂家响应",
	ClaimStatusProcessing: "厂家处理中",
	ClaimStatusOverdue:    "响应超时",
	ClaimStatusInternal:   "自有班组处理中",
	ClaimStatusTakenOver:  "自有班组接手",
	ClaimStatusClosed:     "已闭环",
}

// ComponentLabel 返回质保部件的中文名称。
func ComponentLabel(value string) string {
	if label, ok := componentLabels[value]; ok {
		return label
	}
	return value
}

// PartyLabel 返回责任方的中文名称。
func PartyLabel(value string) string {
	if label, ok := partyLabels[value]; ok {
		return label
	}
	return value
}

// ClaimStatusLabel 返回责任工单状态的中文名称。
func ClaimStatusLabel(value string) string {
	if label, ok := claimStatusLabels[value]; ok {
		return label
	}
	return value
}
