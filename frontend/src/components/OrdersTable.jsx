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
      field: "created_at",
      headerName: "Создан",
      width: 160,
      renderCell: (params) => {
        const value = params.row.created_at;
        return (
          <span>
            {value
              ? new Date(value).toLocaleString("ru-RU")
              : "-"}
          </span>
        );
      }
    },
    {
      field: "completed_at",
      headerName: "Завершён",
      width: 160,
      renderCell: (params) => {
        const nt = params.row.completed_at;
        return (
          <span>
            {nt?.Valid && nt.Time
              ? new Date(nt.Time).toLocaleString("ru-RU")
              : "-"}
          </span>
        );
      }
    },
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
      field: "report",
      headerName: "Отчёт",
      width: 120,
      sortable: false,
      filterable: false,
      renderCell: (params) => (
        <a
          href={`http://localhost:8082/orders/${params.row.id}/report`}
          target="_blank"
          rel="noopener noreferrer"
        >
          PDF отчёт
        </a>
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
    <Box
      sx={{
        height: 'calc(100vh - 160px)',
        width: '100%',
        flexGrow: 1,
        display: 'flex',
        flexDirection: 'column'
      }}
    >
      <DataGrid
        columnHeaderHeight={100}
        rows={orders}
        columns={columns}
        getRowId={(row) => row.id}
        pageSize={10}
        rowsPerPageOptions={[10, 25, 50]}
        disableSelectionOnClick
        sx={{
          '& .MuiDataGrid-cell': {
            justifyContent: 'center',
            textAlign: 'center',
          },
          '& .MuiDataGrid-columnHeader': {
            justifyContent: 'center',
            textAlign: 'center',
          },
          '& .MuiDataGrid-columnHeaderTitle': {
            width: '100%',
            textAlign: 'center',
          }
        }}
      />
    </Box>
  );
}
