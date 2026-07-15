package maintainability

import "testing"

func TestNPathArithmeticSaturates(t *testing.T) {
	if got := npathAdd(npathComplexityLimit-1, 2); got != npathComplexityLimit {
		t.Fatalf("saturated sum = %d", got)
	}
	if got := npathMultiply(50_000, 50_000); got != npathComplexityLimit {
		t.Fatalf("saturated product = %d", got)
	}
}
