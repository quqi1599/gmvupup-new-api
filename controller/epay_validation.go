package controller

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
)

// ZPay accepts unique numeric merchant order IDs of at most 32 characters.
func newEpayTradeNo() (string, error) {
	limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(32), nil)
	value, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%032s", value.String()), nil
}

var epayMoneyPattern = regexp.MustCompile(`^[0-9]{1,12}(\.[0-9]{1,2})?$`)

// Call only after signature verification. Compare against the immutable order
// price, formatted exactly as the checkout request, never the current tariff.
func validEpayOrderPayment(params map[string]string, expectedMoney float64) bool {
	if params["pid"] != operation_setting.EpayId || params["trade_no"] == "" || !epayMoneyPattern.MatchString(params["money"]) {
		return false
	}
	money, err := decimal.NewFromString(params["money"])
	if err != nil || !money.IsPositive() {
		return false
	}
	expected, err := decimal.NewFromString(fmt.Sprintf("%.2f", expectedMoney))
	return err == nil && money.Equal(expected)
}
