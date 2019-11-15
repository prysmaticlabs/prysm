package helper

import "github.com/prysmaticlabs/prysm/shared/params"

// UpdatedGasPrice takes in previous gas price and block size length and return the new gas price
//
// Spec pseudocode definition:
//   def get_updated_gasprice(prev_gasprice: Gwei, length: uint8) -> Gwei:
//    if length > BLOCK_SIZE_TARGET:
//        delta = prev_gasprice * (length - BLOCK_SIZE_TARGET) // BLOCK_SIZE_TARGET // GASPRICE_ADJUSTMENT_COEFFICIENT
//        return min(prev_gasprice + delta, MAX_GASPRICE)
//    else:
//        delta = prev_gasprice * (BLOCK_SIZE_TARGET - length) // BLOCK_SIZE_TARGET // GASPRICE_ADJUSTMENT_COEFFICIENT
//        return max(prev_gasprice, MIN_GASPRICE + delta) - delta
func UpdatedGasPrice(prevGasPrice uint64, length uint64) uint64 {
	bst := params.BeaconConfig().BlockSizeTarget
	if length > bst {
		delta := prevGasPrice * (length - bst) / bst / params.BeaconConfig().GasPriceAdjustmentCoefficient
		min := params.BeaconConfig().MaxGasPrices
		if min > delta+prevGasPrice {
			min = delta + prevGasPrice
		}
		return min
	}
	delta := prevGasPrice * (bst - length) / bst / params.BeaconConfig().GasPriceAdjustmentCoefficient
	max := params.BeaconConfig().MinGasPrices + delta
	if max < prevGasPrice {
		max = prevGasPrice
	}
	return max - delta
}
