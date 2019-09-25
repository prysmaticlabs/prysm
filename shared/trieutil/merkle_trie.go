package trieutil

// NextPowerOf2 returns the next power of 2 >= the input
//
// Spec pseudocode definition:
//   def get_next_power_of_two(x: int) -> int:
//    """
//    Get next power of 2 >= the input.
//    """
//    if x <= 2:
//        return x
//    else:
//        return 2 * get_next_power_of_two((x + 1) // 2)
func NextPowerOf2(n int) int {
	if n <= 2 {
		return n
	}
	return 2 * NextPowerOf2((n+1)/2)
}

// PrevPowerOf2 returns the previous power of 2 >= the input
//
// Spec pseudocode definition:
//   def get_previous_power_of_two(x: int) -> int:
//    """
//    Get the previous power of 2 >= the input.
//    """
//    if x <= 2:
//        return x
//    else:
//        return 2 * get_previous_power_of_two(x // 2)
func PrevPowerOf2(n int) int {
	if n <= 2 {
		return n
	}
	return 2 * PrevPowerOf2(n/2)
}
