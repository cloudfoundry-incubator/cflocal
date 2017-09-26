package mocks

func NewMockFunc() (func(), func() int) {
	var count int
	return func() {
			count++
		}, func() int {
			return count
		}
}
