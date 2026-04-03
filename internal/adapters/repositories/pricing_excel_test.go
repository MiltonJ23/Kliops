package repositories

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// createTestExcelFile writes a "Prix" sheet with the given rows to a temp file
// and returns the file path. Each row is [code, price_string].
func createTestExcelFile(t *testing.T, rows [][]string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test_prices.xlsx")

	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Prix")
	for i, row := range rows {
		cell0, _ := excelize.CoordinatesToCellName(1, i+1)
		cell1, _ := excelize.CoordinatesToCellName(2, i+1)
		f.SetCellValue("Prix", cell0, row[0])
		f.SetCellValue("Prix", cell1, row[1])
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to save test Excel file: %v", err)
	}
	f.Close()
	return path
}

func TestNewExcelPricing_SetsFilePath(t *testing.T) {
	ep := NewExcelPricing("/some/path.xlsx")
	if ep == nil {
		t.Fatal("NewExcelPricing returned nil")
	}
	if ep.FilePath != "/some/path.xlsx" {
		t.Errorf("expected FilePath '/some/path.xlsx', got %q", ep.FilePath)
	}
}

func TestExcelPricing_GetPrice_ArticleFound(t *testing.T) {
	path := createTestExcelFile(t, [][]string{
		{"ART01", "150.5"},
		{"ART02", "45"},
	})
	ep := NewExcelPricing(path)

	price, err := ep.GetPrice(context.Background(), "ART01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 150.5 {
		t.Errorf("expected 150.5, got %f", price)
	}
}

func TestExcelPricing_GetPrice_SecondArticleFound(t *testing.T) {
	path := createTestExcelFile(t, [][]string{
		{"ART01", "100"},
		{"ART02", "45.99"},
	})
	ep := NewExcelPricing(path)

	price, err := ep.GetPrice(context.Background(), "ART02")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 45.99 {
		t.Errorf("expected 45.99, got %f", price)
	}
}

func TestExcelPricing_GetPrice_ArticleNotFound(t *testing.T) {
	path := createTestExcelFile(t, [][]string{
		{"ART01", "100"},
	})
	ep := NewExcelPricing(path)

	_, err := ep.GetPrice(context.Background(), "ART99")
	if err == nil {
		t.Fatal("expected error for unknown article, got nil")
	}
	if !strings.Contains(err.Error(), "ART99") {
		t.Errorf("error message should mention the article code, got: %v", err)
	}
}

func TestExcelPricing_GetPrice_FileNotFound(t *testing.T) {
	ep := NewExcelPricing("/nonexistent/path/prices.xlsx")

	_, err := ep.GetPrice(context.Background(), "ART01")
	if err == nil {
		t.Fatal("expected error when file does not exist, got nil")
	}
	if !strings.Contains(err.Error(), "unable to open the Excel file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExcelPricing_GetPrice_InvalidPriceFormat(t *testing.T) {
	path := createTestExcelFile(t, [][]string{
		{"ART01", "not-a-number"},
	})
	ep := NewExcelPricing(path)

	_, err := ep.GetPrice(context.Background(), "ART01")
	if err == nil {
		t.Fatal("expected error for invalid price format, got nil")
	}
	if !strings.Contains(err.Error(), "price format is invalid") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExcelPricing_GetPrice_EmptyFile(t *testing.T) {
	path := createTestExcelFile(t, [][]string{})
	ep := NewExcelPricing(path)

	_, err := ep.GetPrice(context.Background(), "ART01")
	if err == nil {
		t.Fatal("expected error for empty Excel file, got nil")
	}
}

func TestExcelPricing_GetPrice_RowWithMissingPriceColumn(t *testing.T) {
	// Row has only one column (the code), no price column
	dir := t.TempDir()
	path := filepath.Join(dir, "one_col.xlsx")
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Prix")
	f.SetCellValue("Prix", "A1", "ART01")
	// B1 intentionally left empty - only one column in the row
	f.SaveAs(path)
	f.Close()

	ep := NewExcelPricing(path)
	_, err := ep.GetPrice(context.Background(), "ART01")
	// The implementation checks len(row) >= 2, so ART01 will be skipped
	// and the function will return "article not found"
	if err == nil {
		t.Fatal("expected error when price column is missing, got nil")
	}
}

func TestExcelPricing_GetPrice_ZeroPrice(t *testing.T) {
	path := createTestExcelFile(t, [][]string{
		{"FREE01", "0"},
	})
	ep := NewExcelPricing(path)

	price, err := ep.GetPrice(context.Background(), "FREE01")
	if err != nil {
		t.Fatalf("unexpected error for zero price: %v", err)
	}
	if price != 0.0 {
		t.Errorf("expected 0.0, got %f", price)
	}
}

func TestExcelPricing_GetPrice_WrongSheetName_ReturnsError(t *testing.T) {
	// Create file with sheet "WrongName" instead of "Prix"
	dir := t.TempDir()
	path := filepath.Join(dir, "wrong_sheet.xlsx")
	f := excelize.NewFile()
	// Default sheet is "Sheet1", not "Prix"
	f.SetCellValue("Sheet1", "A1", "ART01")
	f.SetCellValue("Sheet1", "B1", "100")
	f.SaveAs(path)
	f.Close()

	ep := NewExcelPricing(path)
	_, err := ep.GetPrice(context.Background(), "ART01")
	if err == nil {
		t.Fatal("expected error when sheet 'Prix' is missing, got nil")
	}
}