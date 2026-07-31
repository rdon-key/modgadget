package main

func cardputerCoordinate(matrixRow, matrixCol int) (row, col int, ok bool) {
	if matrixRow < 0 || matrixRow >= 7 || matrixCol < 0 || matrixCol >= 8 {
		return 0, 0, false
	}
	col = matrixRow * 2
	if matrixCol > 3 {
		col++
	}
	row = (matrixCol + 4) % 4
	return row, col, true
}
