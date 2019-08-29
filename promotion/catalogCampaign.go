package promotion

import (
	"context"
	"errors"
	offer "nomni/offer-api/models"
	"strconv"
	"time"
)

// Brand（区长）、Member(企划、品牌长)、Channel（企划、品牌长）
type OfferType string

const (
	OfferTypeBrand   OfferType = "brand"
	OfferTypeMember  OfferType = "member"
	OfferTypeChannel OfferType = "channel"
)

type CatalogCampaign struct {
	Id        int64              `json:"id"`
	Code      string             `json:"code"`
	Name      string             `json:"name"`
	Desc      string             `json:"desc"`
	FeeRate   float64            `json:"feeRate"`
	IsStaff   bool               `json:"isStaff"` // 内购（CSL: SaleEvent -> StaffSaleChk）
	Channels  []ChannelCondition `json:"channels"`
	StartAt   time.Time          `json:"startAt"`
	EndAt     time.Time          `json:"endAt"`
	FinalAt   time.Time          `json:"finalAt"` // 延期后的最终结束时间(== CSL：SaleEventEndDate + ExtendSalePermitDateCount)
	RulesetId int64              `json:"rulesetId"`
	Enable    bool               `json:"enable"`
	CreatedAt time.Time          `json:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt"`
}

type CatalogRuleset struct {
	Id           int64              `json:"id"`
	TenantCode   string             `json:"-"`
	TemplateCode string             `json:"templateCode"`
	Type         OfferType          `json:"type"`
	Exclusive    bool               `json:"exclusive"`
	Enable       bool               `json:"enable"`
	Name         string             `json:"name"`
	Action       CatalogAction      `json:"action"`
	Conditions   []CatalogCondition `json:"conditions,omitempty"`
	CreatedAt    time.Time          `json:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt"`
}

type CatalogAction struct {
	DiscountType  DiscountType `json:"discountType" swagger:"enum(to_fixed_price|by_fixed_price|to_percentage)"`
	DiscountValue float64      `json:"discountValue"`
}

type CatalogCondition struct {
	RulesetId int64           `json:"-"`
	SeqNo     int             `json:"seqNo"`
	Cnt       int             `json:"-"`
	Type      string          `json:"type"`
	Comparer  ComparerType    `json:"comparer"`
	Targets   []CatalogTarget `json:"targets"`
}

type CatalogTarget struct {
	RulesetId      int64  `json:"-"`
	ConditionSeqNo int    `json:"-"`
	ConditionType  string `json:"-"`
	Value          string `json:"value"`
}

func CatalogToCSLEvent(ctx context.Context, c CatalogCampaign, ruleSet *CatalogRuleset) (*PromotionEvent, error) {
	var brandCode, storeCode string
	eventType, err := ToCSLOfferType(ruleSet.Type, ruleSet.TemplateCode)
	if err != nil {
		return nil, err
	}
	brand, store, err := getBrandAndStore(ctx, c.Channels)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, errors.New("brand is not exist")
	}
	brandCode = brand.Code
	if store == nil {
		storeCode = ""
	} else {
		storeCode = store.Code
	}

	promotion := &Promotion{
		EventType: eventType,
	}
	promotion.ToCSLDisCount(0, ruleSet.Action.DiscountValue)

	offerNo := strconv.Itoa(int(offer.CampaignTypeCatalog)) + "-" + strconv.FormatInt(c.Id, 10) + "-" + strconv.FormatInt(ruleSet.Id, 10)

	return &PromotionEvent{
		OfferNo:                   offerNo,
		BrandCode:                 brandCode,
		ShopCode:                  storeCode,
		EventTypeCode:             eventType,
		EventName:                 c.Name,
		EventDescription:          c.Desc,
		StartDate:                 c.StartAt,
		EndDate:                   c.FinalAt,
		ExtendSalePermitDateCount: 0,
		NormalSaleRecognitionChk:  promotion.NormalSaleRecognitionChk,
		FeeRate:                   c.FeeRate,
		ApprovalChk:               1,
		InUserID:                  "mslv2.0",
		SaleBaseAmt:               promotion.SaleBaseAmt,
		DiscountBaseAmt:           promotion.DiscountBaseAmt,
		DiscountRate:              promotion.DiscountRate,
		StaffSaleChk:              c.IsStaff,
	}, nil
}
