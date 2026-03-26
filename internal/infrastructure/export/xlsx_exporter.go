package export

import (
	"bytes"
	"context"
	"fmt"

	"github.com/xuri/excelize/v2"

	"supply-chain-simulator/internal/domain"
	"supply-chain-simulator/internal/usecase"
)

type XLSXExporter struct{}

func NewXLSXExporter() XLSXExporter {
	return XLSXExporter{}
}

func (XLSXExporter) ExportSession(_ context.Context, session domain.GameSession, analytics domain.SessionAnalytics) (usecase.ExportedFile, error) {
	file := excelize.NewFile()

	summarySheet := "Summary"
	weeksSheet := "Weeks"
	analyticsSheet := "Analytics"

	file.SetSheetName("Sheet1", summarySheet)
	file.NewSheet(weeksSheet)
	file.NewSheet(analyticsSheet)

	writeSummary(file, summarySheet, session, analytics)
	writeWeeks(file, weeksSheet, session)
	writeAnalytics(file, analyticsSheet, analytics)

	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return usecase.ExportedFile{}, err
	}

	return usecase.ExportedFile{
		FileName:    fmt.Sprintf("session_%s.xlsx", session.ID),
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Content:     buf.Bytes(),
	}, nil
}

func writeSummary(file *excelize.File, sheet string, session domain.GameSession, analytics domain.SessionAnalytics) {
	rows := [][]any{
		{"Session ID", session.ID},
		{"Room ID", session.RoomID},
		{"Status", session.Status},
		{"Scenario ID", session.Scenario.ID},
		{"Weeks Played", analytics.TotalWeeks},
		{"Total Cost", analytics.TotalCost},
		{"Demand Variance", analytics.DemandVariance},
	}

	for rowIndex, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, rowIndex+1)
		_ = file.SetSheetRow(sheet, cell, &row)
	}
}

func writeWeeks(file *excelize.File, sheet string, session domain.GameSession) {
	headers := []any{"Week", "Role", "Incoming Order", "Incoming Goods", "Inventory", "Backlog", "Placed Order", "Shipment", "Weekly Cost"}
	_ = file.SetSheetRow(sheet, "A1", &headers)

	rowIndex := 2
	for _, week := range session.History {
		for _, node := range week.Nodes {
			row := []any{
				week.Week,
				node.Role,
				node.IncomingOrder,
				node.IncomingGoods,
				node.Inventory,
				node.Backlog,
				node.PlacedOrder,
				node.ActualShipment,
				node.WeeklyCost,
			}
			cell, _ := excelize.CoordinatesToCellName(1, rowIndex)
			_ = file.SetSheetRow(sheet, cell, &row)
			rowIndex++
		}
	}
}

func writeAnalytics(file *excelize.File, sheet string, analytics domain.SessionAnalytics) {
	headers := []any{"Role", "Total Cost", "Average Inventory", "Total Backlog", "Total Orders", "Order Variance", "Bullwhip Effect"}
	_ = file.SetSheetRow(sheet, "A1", &headers)

	rowIndex := 2
	for _, node := range analytics.NodeAnalytics {
		row := []any{
			node.Role,
			node.TotalCost,
			node.AverageInventory,
			node.TotalBacklog,
			node.TotalOrders,
			node.OrderVariance,
			analytics.BullwhipEffect[node.Role],
		}
		cell, _ := excelize.CoordinatesToCellName(1, rowIndex)
		_ = file.SetSheetRow(sheet, cell, &row)
		rowIndex++
	}
}
