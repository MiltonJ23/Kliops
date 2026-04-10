package repositories 

import (
	"context"
	"fmt"
	"strconv"
	"github.com/xuri/excelize/v2"
)

type ExcelPricing struct {
	FilePath string 
}

func NewExcelPricing( filepath string) *ExcelPricing {
	return &ExcelPricing{FilePath: filepath}
}

func (e *ExcelPricing) GetPrice(ctx context.Context, codeArticle string) (float64,error) {
	f, fileOpeningError := excelize.OpenFile(e.FilePath)
	if fileOpeningError != nil {
		return 0, fmt.Errorf("unable to open the Excel file : %w",fileOpeningError)
	}
	defer f.Close()

	// we admit the sheet is called "Prix", it holds the columns Column A = Code and Column B = Prix 
	rows, readRowsError := f.GetRows("Prix")
	if readRowsError != nil {
		return 0, fmt.Errorf("unable to parse the excel file Rows: %w",readRowsError)
	}

	for _,row := range rows {
		if len(row) >= 2 && row[0] == codeArticle {
			price, parsePriceError := strconv.ParseFloat(row[1],64)
			if parsePriceError != nil {
				return 0,fmt.Errorf("price format is invalid for %s : %w",codeArticle,parsePriceError)
			}
			return price, nil
		}
	}
	return 0, fmt.Errorf("unable to find article %s in the excel file %s ",codeArticle,e.FilePath)
}