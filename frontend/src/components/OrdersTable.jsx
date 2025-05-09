// src/components/OrdersTable.jsx

import React from "react";
import { DataGrid } from "@mui/x-data-grid";
import {
  Select,
  MenuItem,
  IconButton,
  Typography,
  Box
} from "@mui/material";
import DeleteIcon from "@mui/icons-material/Delete";

export default function OrdersTable({
  orders,
  couriers,
  onDeleteOrder,
  onAssignCourier
}) {
  const columns = [
    { field: "id", headerName: "ID", width: 120 },
    { field: "sender_name", headerName: "Отправитель", width: 150 },
    { field: "recipient_name", headerName: "Получатель", width: 150 },
    { field: "address_from", headerName: "Откуда", width: 200 },
    { field: "address_to", headerName: "Куда", width: 200 },
    {
      field: "status",
      headerName: "Статус",
      width: 120,
      renderCell: (params) => {
        let color = "gray";
        if (params.value === "в пути") color = "blue";
        if (params.value === "завершён") color = "green";
        return (
          <Typography style={{ color, fontWeight: "bold" }}>
            {params.value}
          </Typography>
        );
      }
    },
    {
      field: "courier_id",
      headerName: "Курьер",
      width: 180,
      renderCell: (params) => (
        <Select
          value={params.value || ""}
          size="small"
          onChange={(e) =>
            onAssignCourier(params.row.id, e.target.value)
          }
          displayEmpty
          sx={{ minWidth: 140 }}
        >
          <MenuItem value="">
            <em>Не назначен</em>
          </MenuItem>
          {couriers.map((c) => (
            <MenuItem key={c.id} value={c.id}>
              {c.name}
            </MenuItem>
          ))}
        </Select>
      )
    },
    {
      field: "actions",
      headerName: "Действия",
      width: 100,
      sortable: false,
      filterable: false,
      renderCell: (params) => (
        <IconButton
          color="error"
          onClick={() => onDeleteOrder(params.row.id)}
        >
          <DeleteIcon />
        </IconButton>
      )
    }
  ];

  return (
    <Box sx={{ height: 600, width: "100%" }}>
      <DataGrid
        rows={orders}
        columns={columns}
        getRowId={(row) => row.id}
        pageSize={10}
        rowsPerPageOptions={[10, 25, 50]}
        disableSelectionOnClick
      />
    </Box>
  );
}
