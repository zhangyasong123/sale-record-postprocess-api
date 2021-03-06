package saleRecordFee

import (
	"context"
	"errors"
	"nhub/sale-record-postprocess-api/models"
	"nhub/sale-record-postprocess-api/promotion"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func (PostSaleRecordFee) MakePostSaleRecordFeesEntity(ctx context.Context, a models.SaleRecordEvent) ([]PostSaleRecordFee, error) {
	var postSaleRecordFees []PostSaleRecordFee
	var eventFeeRate, appliedFeeRate, feeAmount, itemFeeRate float64
	var eventTypeCode, itemIds string

	appliedFeeRate = 0
	eventTypeCode = ""
	eventFeeRate = 0
	var cartOffers []models.CartOffer
	var promotionEvents []*promotion.PromotionEvent

	for _, cartOffer := range a.CartOffers {
		if cartOffer.CouponNo == "" && cartOffer.OfferNo != "" {
			promotionEvent, err := promotion.GetByNo(ctx, cartOffer.OfferNo)
			if err != nil {
				logrus.WithField("Error", err).Info("GetPromotionEvent error")
				return nil, err
			}
			eventTypeCode = promotionEvent.EventTypeCode
			if eventTypeCode == "01" || eventTypeCode == "02" || eventTypeCode == "03" {
				if promotionEvent.FeeRate <= 0 {
					return nil, errors.New("活动扣率不能为0！")
				}
				cartOffers = append(cartOffers, cartOffer)
				promotionEvents = append(promotionEvents, promotionEvent)
			}
		}
	}
	for _, assortedSaleRecordDtl := range a.AssortedSaleRecordDtlList {
		eventFeeRate = 0
		appliedFeeRate = 0
		feeAmount = 0
		for _, cartOffer := range cartOffers {
			for _, promotionEvent := range promotionEvents {
				if promotionEvent.OfferNo == cartOffer.OfferNo {
					itemIds = cartOffer.ItemIds
					result := strings.Index(itemIds+",", strconv.FormatInt(assortedSaleRecordDtl.OrderItemId, 10)+",")
					if result != -1 {
						eventFeeRate = promotionEvent.FeeRate
						appliedFeeRate = promotionEvent.FeeRate
						break
					}
				}
			}
		}
		itemFeeRate = assortedSaleRecordDtl.FeeRate
		// eventFeeRate 优先级大于 itemFeeRate
		if appliedFeeRate == 0 && itemFeeRate > 0 {
			appliedFeeRate = itemFeeRate
		}
		sellingAmt := assortedSaleRecordDtl.TotalPrice.ListPrice - assortedSaleRecordDtl.DistributedPrice.TotalDistributedCartOfferPrice -
			assortedSaleRecordDtl.DistributedPrice.TotalDistributedItemOfferPrice - assortedSaleRecordDtl.MileagePrice

		feeAmount = GetToFixedPrice(sellingAmt*appliedFeeRate/100, "feeAmount")
		postSaleRecordFees = append(
			postSaleRecordFees,
			PostSaleRecordFee{
				TransactionId:          a.TransactionId,
				TransactionDtlId:       assortedSaleRecordDtl.Id,
				OrderId:                a.OrderId,
				OrderItemId:            assortedSaleRecordDtl.OrderItemId,
				RefundId:               a.RefundId,
				RefundItemId:           assortedSaleRecordDtl.RefundItemId,
				CustomerId:             a.CustomerId,
				StoreId:                a.StoreId,
				TotalSalePrice:         assortedSaleRecordDtl.TotalPrice.SalePrice,
				TotalPaymentPrice:      assortedSaleRecordDtl.TotalPrice.ListPrice - assortedSaleRecordDtl.DistributedPrice.TotalDistributedCartOfferPrice - assortedSaleRecordDtl.DistributedPrice.TotalDistributedItemOfferPrice,
				Mileage:                assortedSaleRecordDtl.Mileage,
				MileagePrice:           assortedSaleRecordDtl.MileagePrice,
				ItemFeeRate:            itemFeeRate,
				ItemFee:                assortedSaleRecordDtl.ItemFee,
				EventFeeRate:           eventFeeRate,
				AppliedFeeRate:         appliedFeeRate,
				FeeAmount:              feeAmount,
				TransactionChannelType: a.TransactionChannelType,
			},
		)
	}
	return postSaleRecordFees, nil
}
