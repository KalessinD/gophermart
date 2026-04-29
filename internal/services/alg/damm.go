package alg

import (
	"strings"
)

// Матрица Дамма (Quasigroup)
var dammTable = [10][10]int{
	{0, 3, 1, 7, 5, 9, 8, 6, 4, 2},
	{7, 0, 9, 2, 1, 5, 4, 8, 6, 3},
	{4, 2, 0, 6, 8, 7, 1, 3, 5, 9},
	{1, 7, 5, 0, 9, 8, 3, 4, 2, 6},
	{6, 1, 2, 3, 0, 4, 5, 9, 7, 8},
	{3, 6, 7, 4, 2, 0, 9, 5, 8, 1},
	{5, 8, 6, 9, 7, 2, 0, 1, 3, 4},
	{8, 9, 4, 5, 3, 6, 2, 0, 1, 7},
	{9, 4, 3, 8, 6, 1, 7, 2, 0, 5},
	{2, 5, 8, 1, 4, 3, 6, 7, 0, 9},
}

// IsValidDamm проверяет полное число (включая контрольную цифру)
func IsValidDamm(number string) bool {
	// Очистка пробелов и дефисов
	cleanNumber := strings.ReplaceAll(number, " ", "")
	cleanNumber = strings.ReplaceAll(cleanNumber, "-", "")

	// Число должно иметь хотя бы одну цифру
	if len(cleanNumber) == 0 {
		return false
	}

	// Вычисляем промежуточное значение
	interim := 0
	for _, r := range cleanNumber {
		if r < '0' || r > '9' {
			return false
		}
		digit := int(r - '0')
		interim = dammTable[interim][digit]
	}

	// Если результат 0, то число валидно
	return interim == 0
}
