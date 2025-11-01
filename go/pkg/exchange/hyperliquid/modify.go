package hyperliquid

import (
	"fmt"
	"strings"

	"nof0-api/pkg/exchange"
)

// ModifyOrderRequest identifies a resting order and the updated parameters.
// Exactly one of Oid or Cloid must be provided.
type ModifyOrderRequest struct {
	Oid   *int64
	Cloid string
	Order exchange.Order
}

func buildModifyPayload(req ModifyOrderRequest) (modifyPayload, error) {
	hasOid := req.Oid != nil
	cloid := strings.TrimSpace(req.Cloid)
	hasCloid := cloid != ""

	if hasOid == hasCloid {
		return modifyPayload{}, fmt.Errorf("hyperliquid: modify request must specify exactly one of oid or cloid")
	}

	payload, err := convertOrder(req.Order)
	if err != nil {
		return modifyPayload{}, err
	}

	var identifier interface{}
	if hasOid {
		if *req.Oid <= 0 {
			return modifyPayload{}, fmt.Errorf("hyperliquid: modify oid must be positive")
		}
		identifier = *req.Oid
	} else {
		if len(cloid) > 128 {
			return modifyPayload{}, fmt.Errorf("hyperliquid: modify cloid exceeds 128 characters")
		}
		identifier = cloid
	}

	return modifyPayload{
		Oid:   identifier,
		Order: payload,
	}, nil
}

func buildModifyAction(req ModifyOrderRequest) (modifyAction, error) {
	payload, err := buildModifyPayload(req)
	if err != nil {
		return modifyAction{}, err
	}
	return modifyAction{
		Type:  ActionTypeModify,
		Oid:   payload.Oid,
		Order: payload.Order,
	}, nil
}

func buildBatchModifyAction(requests []ModifyOrderRequest) (batchModifyAction, error) {
	if len(requests) == 0 {
		return batchModifyAction{}, fmt.Errorf("hyperliquid: at least one modify request required")
	}
	modifies := make([]modifyPayload, len(requests))
	for i, req := range requests {
		payload, err := buildModifyPayload(req)
		if err != nil {
			return batchModifyAction{}, fmt.Errorf("modify[%d]: %w", i, err)
		}
		modifies[i] = payload
	}
	return batchModifyAction{
		Type:     ActionTypeBatchModify,
		Modifies: modifies,
	}, nil
}
