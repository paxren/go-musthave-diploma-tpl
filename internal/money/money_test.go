package money

import (
	"testing"
)

func TestKopecksToRubles(t *testing.T) {
	tests := []struct {
		name     string
		kopecks  uint64
		expected float64
	}{
		{"0 копеек", 0, 0.0},
		{"1 копейка", 1, 0.01},
		{"100 копеек", 100, 1.0},
		{"101 копейка", 101, 1.01},
		{"999 копеек", 999, 9.99},
		{"1000 копеек", 1000, 10.0},
		{"1505 копеек", 1505, 15.05},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := KopecksToRubles(tt.kopecks)
			if result != tt.expected {
				t.Errorf("KopecksToRubles(%d) = %.2f; ожидалось %.2f",
					tt.kopecks, result, tt.expected)
			}
		})
	}
}

func TestRublesToKopecks(t *testing.T) {
	tests := []struct {
		name     string
		rubles   float64
		expected uint64
	}{
		{"0 рублей", 0.0, 0},
		{"1 рубль", 1.0, 100},
		{"1.01 рубль", 1.01, 101},
		{"0.01 рубль", 0.01, 1},
		{"9.99 рублей", 9.99, 999},
		{"10.0 рублей", 10.0, 1000},
		{"15.05 рублей", 15.05, 1505},
		{"50.50 рублей", 50.50, 5050},
		{"100.99 рублей", 100.99, 10099},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RublesToKopecks(tt.rubles)
			if result != tt.expected {
				t.Errorf("RublesToKopecks(%.3f) = %d; ожидалось %d",
					tt.rubles, result, tt.expected)
			}
		})
	}
}

func TestAccrualToKopecks(t *testing.T) {
	tests := []struct {
		name         string
		accrualValue float64
		expected     uint64
		shouldError  bool
	}{
		{"0 рублей", 0, 0, false},
		{"1 рубль", 1, 100, false},
		{"10 рублей", 10, 1000, false},
		{"100 рублей", 100, 10000, false},
		{"1000 рублей", 1000, 100000, false},
		{"10000 рублей", 10000, 1000000, false},
		{"100000 рублей", 100000, 10000000, false},
		{"1000000 рублей", 1000000, 100000000, false},
		{"123.45 рублей", 123.45, 12345, false},
		{"Максимальное безопасное значение", 18446744073709551, 1844674407370955100, false},
		{"Значение, вызывающее переполнение", 18446744073709552, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AccrualToKopecks(tt.accrualValue)

			if tt.shouldError {
				if err == nil {
					t.Errorf("AccrualToKopecks(%f) ожидал ошибку, но ошибки не было", tt.accrualValue)
				}
				if err != ErrOverflow {
					t.Errorf("AccrualToKopecks(%f) вернул ошибку %v, ожидалось %v",
						tt.accrualValue, err, ErrOverflow)
				}
				return
			}

			if err != nil {
				t.Errorf("AccrualToKopecks(%f) вернул неожиданную ошибку: %v",
					tt.accrualValue, err)
				return
			}

			if result != tt.expected {
				t.Errorf("AccrualToKopecks(%f) = %d; ожидалось %d",
					tt.accrualValue, result, tt.expected)
			}
		})
	}
}

func TestFormatRubles(t *testing.T) {
	tests := []struct {
		name     string
		kopecks  uint64
		expected string
	}{
		{"0 копеек", 0, "0.00"},
		{"1 копейка", 1, "0.01"},
		{"100 копеек", 100, "1.00"},
		{"101 копейка", 101, "1.01"},
		{"999 копеек", 999, "9.99"},
		{"1000 копеек", 1000, "10.00"},
		{"1505 копеек", 1505, "15.05"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatRubles(tt.kopecks)
			if result != tt.expected {
				t.Errorf("FormatRubles(%d) = %s; ожидалось %s",
					tt.kopecks, result, tt.expected)
			}
		})
	}
}
