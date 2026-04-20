package testfixture

// ID is a type alias for string.
type ID = string

// Status constants using iota.
const (
	StatusPending = iota
	StatusActive
	StatusDone
)

// Item is a struct with an embedded Config.
type Item struct {
	Config
	ID   ID
	Tags []string
}

// Sum adds all provided integers (variadic).
func Sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// MinMax returns the minimum and maximum of a slice (named returns).
func MinMax(nums []int) (min, max int) {
	if len(nums) == 0 {
		return 0, 0
	}
	min, max = nums[0], nums[0]
	for _, n := range nums[1:] {
		if n < min {
			min = n
		}
		if n > max {
			max = n
		}
	}
	return
}
