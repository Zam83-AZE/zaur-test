package collector

import "testing"

func TestIsCI(t *testing.T) {
	result := IsCI()
	t.Logf("IsCI: %v", result)
}

func TestSkipOnCI(t *testing.T) {
	if IsCI() {
		SkipOnCI(t)
		t.Error("should have skipped")
	}
}
