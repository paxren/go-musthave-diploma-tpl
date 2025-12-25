package money

import (
	"errors"
	"fmt"
	"math"
)

var ErrOverflow = errors.New("переполнение при конвертации денежных значений")

// KopecksToRubles конвертирует копейки в рубли
func KopecksToRubles(kopecks uint64) float64 {
	return float64(kopecks) / 100.0
}

// RublesToKopecks конвертирует рубли в копейки с корректным округлением
func RublesToKopecks(rubles float64) uint64 {
	// Используем math.Round для корректного округления до ближайшего целого
	return uint64(math.Round(rubles * 100))
}

// AccrualToKopecks конвертирует значение от accrual системы (в рублях) в копейки
// с проверкой на переполнение
func AccrualToKopecks(accrualRubles float64) (uint64, error) {
	// Проверяем на переполнение: max uint64 / 100
	const maxSafeValue = 18446744073709551 // math.MaxUint64 / 100

	if accrualRubles > maxSafeValue {
		return 0, ErrOverflow
	}

	return uint64(accrualRubles * 100), nil
}

// FormatRubles форматирует копейки в строку с рублями
func FormatRubles(kopecks uint64) string {
	rubles := float64(kopecks) / 100.0
	return fmt.Sprintf("%.2f", rubles)
}
