package saleRecordFee

import (
	"context"
	"math"
	"nhub/sale-record-postprocess-api/models"
	"nhub/sale-record-postprocess-api/promotion"
	"strings"

	"github.com/sirupsen/logrus"
)

func (PostSaleRecordFee) MakePostSaleRecordFeesEntity(ctx context.Context, a models.SaleRecordEvent) ([]PostSaleRecordFee, error) {
	var postSaleRecordFees []PostSaleRecordFee
	var eventFeeRate, appliedFeeRate, feeAmount, itemFeeRate float64
	var eventTypeCode, itemCodes string

	appliedFeeRate = 0
	eventTypeCode = ""
	eventFeeRate = 0
	var cartOffers []models.CartOffer
	for _, cartOffer := range a.CartOffers {
		if eventFeeRate != 0 {
			continue
		}
		if cartOffer.CouponNo == "" && cartOffer.OfferNo != "" {
			promotionEvent, err := promotion.GetByNo(ctx, cartOffer.OfferNo)
			if err != nil {
				logrus.WithField("Error", err).Info("GetPromotionEvent error")
				return nil, err
			}
			eventTypeCode = promotionEvent.EventTypeCode
			if eventTypeCode == "01" || eventTypeCode == "02" || eventTypeCode == "03" {
				cartOffers = append(cartOffers, cartOffer)
				eventFeeRate = promotionEvent.FeeRate
				if eventFeeRate <= 0 {
					postFailCreateSaleFee := &PostFailCreateSaleFee{TransactionId: a.TransactionId, IsProcessed: false}
					has, _, err := postFailCreateSaleFee.Get(ctx)
					if err != nil {
						return nil, err
					}
					if !has {
						if err := postFailCreateSaleFee.Save(ctx); err != nil {
							return nil, err
						}
					}
					return nil, nil
				}
			}
		}
	}

	for _, assortedSaleRecordDtl := range a.AssortedSaleRecordDtlList {
		appliedFeeRate = 0
		feeAmount = 0
		for _, cartOffer := range cartOffers {
			itemCodes = cartOffer.ItemCodes
			result := strings.Index(itemCodes+",", assortedSaleRecordDtl.ItemCode+",")
			if result != -1 {
				appliedFeeRate = eventFeeRate
				break
			}
		}
		itemFeeRate = assortedSaleRecordDtl.FeeRate
		// eventFeeRate 优先级大于 itemFeeRate
		if appliedFeeRate == 0 && itemFeeRate > 0 {
			appliedFeeRate = itemFeeRate
		}
		// total_list_price -  total_distributed_cart_offer_price - total_distributed_item_offer_price - (total_distributed_payment_price - distributed_cash_price)
		sellingAmt := assortedSaleRecordDtl.TotalPrice.ListPrice - assortedSaleRecordDtl.DistributedPrice.TotalDistributedCartOfferPrice -
			assortedSaleRecordDtl.DistributedPrice.TotalDistributedItemOfferPrice - assortedSaleRecordDtl.MileagePrice
		// SellingAmt-(floor(((SellingAmt-SellingAmt*FeeRate/100)*1/0.01))*0.01)
		feeAmount = sellingAmt - (math.Floor(((sellingAmt - sellingAmt*appliedFeeRate/100) / 0.01)) * 0.01)
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
