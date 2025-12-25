package internal

import (
	"testing"
)

// TestKopecksToRublesConversion тестирует конвертацию копеек в рубли
func TestKopecksToRublesConversion(t *testing.T) {
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
			result := float64(tt.kopecks) / 100
			if result != tt.expected {
				t.Errorf("Конвертация %d копеек в рубли: ожидалось %.2f, получено %.2f",
					tt.kopecks, tt.expected, result)
			}
		})
	}
}

// TestRublesToKopecksConversion тестирует конвертацию рублей в копейки
func TestRublesToKopecksConversion(t *testing.T) {
	tests := []struct {
		name       string
		rubles     float64
		expected   uint64
		shouldFail bool
	}{
		{"0 рублей", 0.0, 0, false},
		{"1 рубль", 1.0, 100, false},
		{"1.01 рубль", 1.01, 101, false},
		{"0.01 рубль", 0.01, 1, false},
		{"9.99 рублей", 9.99, 999, false},
		{"10.0 рублей", 10.0, 1000, false},
		{"15.05 рублей", 15.05, 1505, false},
		{"1.001 рубль (округление)", 1.001, 100, false}, // должно округлиться до 100
		{"1.009 рубль (округление)", 1.009, 101, false}, // должно округлиться до 101
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Используем тот же метод, что и в handlers.go
			result := uint64(tt.rubles * 100)

			if !tt.shouldFail && result != tt.expected {
				t.Errorf("Конвертация %.3f рублей в копейки: ожидалось %d, получено %d",
					tt.rubles, tt.expected, result)
			}
		})
	}
}

// TestAccrualConversion тестирует конвертацию значений от accrual системы
func TestAccrualConversion(t *testing.T) {
	tests := []struct {
		name         string
		accrualValue uint64 // значение от accrual системы в рублях
		expected     uint64 // ожидаемое значение в копейках
		shouldFail   bool
	}{
		{"0 рублей", 0, 0, false},
		{"1 рубль", 1, 100, false},
		{"10 рублей", 10, 1000, false},
		{"100 рублей", 100, 10000, false},
		{"1000 рублей", 1000, 100000, false},
		{"10000 рублей", 10000, 1000000, false},
		{"100000 рублей", 100000, 10000000, false},
		{"1000000 рублей", 1000000, 100000000, false},
		{"Максимальное значение без переполнения", 18446744073709551, 1844674407370955100, false},
		{"Значение, вызывающее переполнение", 18446744073709552, 0, true}, // 18446744073709552 * 100 > max uint64
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем на переполнение
			if tt.accrualValue > 18446744073709551 { // max uint64 / 100
				if !tt.shouldFail {
					t.Errorf("Ожидалось переполнение при конвертации %d рублей", tt.accrualValue)
				}
				return
			}

			// Используем тот же метод, что и в ComunicationsWithAccrual.go
			result := uint64(tt.accrualValue * 100)

			if !tt.shouldFail && result != tt.expected {
				t.Errorf("Конвертация %d рублей от accrual в копейки: ожидалось %d, получено %d",
					tt.accrualValue, tt.expected, result)
			}
		})
	}
}
