package export

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestBuildDisclaimer(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	params := testParams()
	exportCode := "EXP-20260404-120000-1234"

	err := buildDisclaimer(f, params, exportCode)
	require.NoError(t, err)

	// Verify sheet exists
	sheetIdx, err := f.GetSheetIndex("Tuyên bố miễn trừ")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, sheetIdx, 0, "disclaimer sheet should exist")

	// Verify title content
	val, err := f.GetCellValue("Tuyên bố miễn trừ", "A1")
	require.NoError(t, err)
	assert.Contains(t, val, "KIENLONGBANK")
	assert.Contains(t, val, "TÀI LIỆU MẬT")

	// Verify export info rows exist
	val, err = f.GetCellValue("Tuyên bố miễn trừ", "A3")
	require.NoError(t, err)
	assert.Equal(t, "Người xuất", val)

	val, err = f.GetCellValue("Tuyên bố miễn trừ", "A4")
	require.NoError(t, err)
	assert.Equal(t, "Chức danh", val)

	val, err = f.GetCellValue("Tuyên bố miễn trừ", "A5")
	require.NoError(t, err)
	assert.Equal(t, "Thời điểm", val)

	val, err = f.GetCellValue("Tuyên bố miễn trừ", "A6")
	require.NoError(t, err)
	assert.Equal(t, "Khoảng dữ liệu", val)

	val, err = f.GetCellValue("Tuyên bố miễn trừ", "A7")
	require.NoError(t, err)
	assert.Equal(t, "Mã xuất", val)

	// Verify export code value
	val, err = f.GetCellValue("Tuyên bố miễn trừ", "B7")
	require.NoError(t, err)
	assert.Equal(t, exportCode, val)

	// Verify rules section header
	val, err = f.GetCellValue("Tuyên bố miễn trừ", "A9")
	require.NoError(t, err)
	assert.Equal(t, "Quy định bảo mật:", val)

	// Verify Vietnamese diacritics in rules
	val, err = f.GetCellValue("Tuyên bố miễn trừ", "A10")
	require.NoError(t, err)
	assert.Contains(t, val, "nghiêm cấm")

	val, err = f.GetCellValue("Tuyên bố miễn trừ", "A11")
	require.NoError(t, err)
	assert.Contains(t, val, "ủy quyền")
}
