package main

import "testing"

func TestCardputerCoordinateExamples(t *testing.T) {
	tests := []struct {
		matrixRow int
		matrixCol int
		row       int
		col       int
	}{
		{0, 0, 0, 0},
		{0, 4, 0, 1},
		{1, 0, 0, 2},
		{6, 7, 3, 13},
	}
	for _, test := range tests {
		row, col, ok := cardputerCoordinate(test.matrixRow, test.matrixCol)
		if !ok || row != test.row || col != test.col {
			t.Fatalf("matrix (%d,%d) = (%d,%d,%v), want (%d,%d,true)", test.matrixRow, test.matrixCol, row, col, ok, test.row, test.col)
		}
	}
}

func TestCardputerCoordinateCoversPhysicalMatrix(t *testing.T) {
	var seen [4][14]bool
	count := 0
	for matrixRow := 0; matrixRow < 7; matrixRow++ {
		for matrixCol := 0; matrixCol < 8; matrixCol++ {
			row, col, ok := cardputerCoordinate(matrixRow, matrixCol)
			if !ok {
				t.Fatalf("matrix (%d,%d) was rejected", matrixRow, matrixCol)
			}
			if seen[row][col] {
				t.Fatalf("physical (%d,%d) is duplicated", row, col)
			}
			seen[row][col] = true
			count++
		}
	}
	if count != 56 {
		t.Fatalf("mapped positions = %d", count)
	}
	for row := range seen {
		for col := range seen[row] {
			if !seen[row][col] {
				t.Fatalf("physical (%d,%d) was not mapped", row, col)
			}
		}
	}
}

func TestCardputerCoordinateRejectsOutOfRange(t *testing.T) {
	tests := [][2]int{{-1, 0}, {7, 0}, {0, -1}, {0, 8}}
	for _, test := range tests {
		if row, col, ok := cardputerCoordinate(test[0], test[1]); ok {
			t.Fatalf("matrix (%d,%d) = (%d,%d), want invalid", test[0], test[1], row, col)
		}
	}
}
