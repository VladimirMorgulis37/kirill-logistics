import React from "react";
import { DataGrid } from "@mui/x-data-grid";
import { Typography, Box } from "@mui/material";

export default function CourierStatsTable({ couriers }) {
  const columns = [
    { field: "courier_id", headerName: "ID курьера", width: 150 },
    { field: "courier_name", headerName: "Имя", width: 180 },
    {
      field: "completed_orders",
      headerName: "Завершённые заказы",
      width: 180,
      type: "number"
    },
    {
      field: "total_revenue",
      headerName: "Выручка (₽)",
      width: 160,
      type: "number",
      renderCell: (params) => (
        <Typography fontWeight="bold">
          {params.value.toFixed(2)}
        </Typography>
      )
    },
    {
      field: "average_delivery_time_sec",
      headerName: "Сред. время (мин)",
      width: 180,
      type: "number",
      renderCell: (params) => {
        const minutes = (params.value / 60).toFixed(1);
        return <span>{minutes} мин</span>;
      }
    }
  ];

  return (
    <Box sx={{ height: 600, width: "100%" }}>
      <DataGrid
        rows={couriers}
        columns={columns}
        getRowId={(row) => row.courier_id}
        pageSize={10}
        rowsPerPageOptions={[10, 25, 50]}
        disableSelectionOnClick
        columnHeaderHeight={100}
      />
    </Box>
  );
}
