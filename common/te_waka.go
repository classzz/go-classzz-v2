package common

import (
	"math/big"
)

var (
	baseUnit  = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	fbaseUnit = new(big.Float).SetFloat64(float64(baseUnit.Int64()))
)

func ToCzz(val *big.Int) *big.Float {
	return new(big.Float).Quo(new(big.Float).SetInt(val), fbaseUnit)
}
