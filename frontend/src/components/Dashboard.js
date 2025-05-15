// src/components/Dashboard.js (с улучшением Material UI)

import React, { useState, useEffect } from "react";
import {
  AppBar, Toolbar, Typography, Button, Container, Paper,
  TextField, Select, MenuItem, FormControl, InputLabel, Grid, Box
} from "@mui/material";
import { HashRouter as Router, Routes, Route, NavLink, Navigate } from "react-router-dom";
import API_URLS from "../config";
import CourierMap from "./CourierMap";
import OrdersTable from "./OrdersTable";
import CourierStatsTable from "./CourierStatsTable";

export default function Dashboard({ token, onLogout }) {
  const [orders, setOrders] = useState([]);
  const [couriers, setCouriers] = useState([]);
  const [sel, setSel] = useState("");
  const [courierStats, setCourierStats] = useState([]);

  const [newOrder, setNewOrder] = useState({
    senderName: "",
    email: "",
    recipientName: "",
    addressFrom: "",
    addressTo: "",
    weight: "",
    length: "",
    width: "",
    height: "",
    urgency: "1",
  });

  const [newCourier, setNewCourier] = useState({
    name: "",
    phone: "",
    vehicle: "foot",
    lat: 55.7558,
    lng: 37.6176
  });

  useEffect(() => {
    fetch(`${API_URLS.orders}/couriers`, { headers: { Authorization: `Bearer ${token}` } })
      .then(res => res.ok ? res.json() : [])
      .then(setCouriers)
      .catch(() => setCouriers([]));

    fetch(`${API_URLS.orders}/orders`, { headers: { Authorization: `Bearer ${token}` } })
      .then(res => res.json())
      .then(data => {
        setOrders(data);
        if (data.length) setSel(data[0].id);
      })
      .catch(() => setOrders([]));

    fetch(`${API_URLS.analytics}/analytics/couriers`, { headers: { Authorization: `Bearer ${token}` } })
      .then(res => res.ok ? res.json() : [])
      .then(setCourierStats)
      .catch(() => setCourierStats([]));
  }, [token]);

  const handleInputChange = e => {
    const { name, value } = e.target;
    setNewOrder(prev => ({ ...prev, [name]: value }));
  };

  const handleCourierChange = e => {
    const { name, value } = e.target;
    setNewCourier(prev => ({ ...prev, [name]: value }));
  };

  const handleAddOrder = e => {
    e.preventDefault();
    const payload = {
      sender_name: newOrder.senderName,
      email: newOrder.email,
      recipient_name: newOrder.recipientName,
      address_from: newOrder.addressFrom,
      address_to: newOrder.addressTo,
      weight: parseFloat(newOrder.weight),
      length: parseFloat(newOrder.length),
      width: parseFloat(newOrder.width),
      height: parseFloat(newOrder.height),
      urgency: parseInt(newOrder.urgency, 10),
    };
    fetch(`${API_URLS.orders}/orders`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(payload),
    })
      .then(res => res.json())
      .then(o => {
        setOrders(prev => [...prev, o]);
        setSel(o.id);
        setNewOrder({ senderName: "", email: "", recipientName: "", addressFrom: "", addressTo: "", weight: "", length: "", width: "", height: "", urgency: "1" });
      })
      .catch(console.error);
  };

  const handleAddCourier = e => {
    e.preventDefault();
    const payload = {
      name: newCourier.name,
      phone: newCourier.phone,
      vehicle_type: newCourier.vehicle,
      latitude: parseFloat(newCourier.lat),
      longitude: parseFloat(newCourier.lng),
      status: "available",
      active_order_id: "",
    };
    fetch(`${API_URLS.orders}/couriers`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(payload),
    })
      .then(res => res.json())
      .then(c => {
        setCouriers(prev => [...prev, c]);
        setNewCourier({ name: "", phone: "", vehicle: "foot", lat: 55.7558, lng: 37.6176 });
      })
      .catch(console.error);
  };

  const handleDeleteOrder = id => {
    fetch(`${API_URLS.orders}/orders/${id}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(() => {
        setOrders(prev => prev.filter(o => o.id !== id));
        if (sel === id) setSel(orders[0]?.id || "");
      })
      .catch(console.error);
  };

  const handleAssignCourier = (orderId, courierId) => {
    fetch(`${API_URLS.orders}/orders/${orderId}/assign-courier`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ courier_id: courierId }),
    })
      .then(() => {
        setOrders(prev => prev.map(o => o.id === orderId ? { ...o, courier_id: courierId } : o));
        if (sel === orderId) setSel(orderId);
      })
      .catch(console.error);
  };

  return (
    <Router>
      <AppBar position="static">
        <Toolbar>
          <Typography variant="h6" sx={{ flexGrow: 1 }}>Панель управления</Typography>
          <Button color="inherit" component={NavLink} to="/add-order">Добавить заказ</Button>
          <Button color="inherit" component={NavLink} to="/add-courier">Добавить курьера</Button>
          <Button color="inherit" component={NavLink} to="/orders">Список заказов</Button>
          <Button color="inherit" component={NavLink} to="/tracking">Отслеживание</Button>
          <Button color="inherit" component={NavLink} to="/couriers-stats">Аналитика</Button>
          <Button variant="contained" color="secondary" sx={{ ml: 2 }} onClick={onLogout}>Выйти</Button>
        </Toolbar>
      </AppBar>

      <Container sx={{ mt: 4 }}>
        <Routes>
          <Route path="/" element={<Navigate to="/add-order" replace />} />

          <Route path="/add-order" element={
            <Paper sx={{ p: 3 }}>
              <Typography variant="h5" gutterBottom>Добавить заказ</Typography>
              <Box component="form" onSubmit={handleAddOrder}>
                <Grid container spacing={2}>
                  {["senderName", "email", "recipientName", "addressFrom", "addressTo", "weight", "length", "width", "height"].map(field => (
                    <Grid item xs={12} sm={6} key={field}>
                      <TextField
                        label={field === "email" ? "Email отправителя" : field.replace(/([A-Z])/g, " $1")}
                        name={field}
                        type={field === "email" ? "email" : "text"}
                        value={newOrder[field]}
                        onChange={handleInputChange}
                        required
                        fullWidth
                      />
                    </Grid>
                  ))}
                  <Grid item xs={12} sm={6}>
                    <FormControl fullWidth>
                      <InputLabel>Срочность</InputLabel>
                      <Select name="urgency" value={newOrder.urgency} label="Срочность" onChange={handleInputChange}>
                        <MenuItem value="1">Стандартная</MenuItem>
                        <MenuItem value="2">Экспресс</MenuItem>
                      </Select>
                    </FormControl>
                  </Grid>
                </Grid>
                <Button type="submit" variant="contained" sx={{ mt: 3 }}>Добавить</Button>
              </Box>
            </Paper>
          } />

          <Route path="/add-courier" element={
            <Paper sx={{ p: 3 }}>
              <Typography variant="h5" gutterBottom>Добавить курьера</Typography>
              <Box component="form" onSubmit={handleAddCourier}>
                <Grid container spacing={2}>
                  <Grid item xs={12} sm={6}>
                    <TextField label="Имя" name="name" fullWidth required value={newCourier.name} onChange={handleCourierChange} />
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <TextField label="Телефон" name="phone" fullWidth value={newCourier.phone} onChange={handleCourierChange} />
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <FormControl fullWidth>
                      <InputLabel>Тип транспорта</InputLabel>
                      <Select name="vehicle" value={newCourier.vehicle} label="Тип транспорта" onChange={handleCourierChange}>
                        <MenuItem value="foot">Пеший</MenuItem>
                        <MenuItem value="bike">Велосипед</MenuItem>
                        <MenuItem value="car">Автомобиль</MenuItem>
                      </Select>
                    </FormControl>
                  </Grid>
                  <Grid item xs={12} sm={3}>
                    <TextField label="Широта" name="lat" fullWidth type="number" value={newCourier.lat} onChange={handleCourierChange} />
                  </Grid>
                  <Grid item xs={12} sm={3}>
                    <TextField label="Долгота" name="lng" fullWidth type="number" value={newCourier.lng} onChange={handleCourierChange} />
                  </Grid>
                </Grid>
                <Button type="submit" variant="contained" sx={{ mt: 3 }}>Добавить курьера</Button>
              </Box>
            </Paper>
          } />

          <Route path="/orders" element={
            <>
              <Typography variant="h5" gutterBottom>Список заказов</Typography>
              <OrdersTable
                orders={orders}
                couriers={couriers}
                onDeleteOrder={handleDeleteOrder}
                onAssignCourier={handleAssignCourier}
              />
            </>
          } />

          <Route path="/tracking" element={
            <Paper sx={{ p: 3 }}>
              <Typography variant="h5" gutterBottom>Отслеживание</Typography>
              {orders.length ? (
                <FormControl fullWidth sx={{ mb: 2 }}>
                  <InputLabel>Выберите заказ</InputLabel>
                  <Select value={sel} onChange={e => setSel(e.target.value)} label="Выберите заказ">
                    {orders.map(o => (
                      <MenuItem key={o.id} value={o.id}>{o.sender_name} → {o.recipient_name}</MenuItem>
                    ))}
                  </Select>
                </FormControl>
              ) : <Typography>Нет заказов</Typography>}
              {sel && <CourierMap orderId={sel} token={token} />}
            </Paper>
          } />

          <Route path="/couriers-stats" element={
            <Paper sx={{ p: 3 }}>
              <Typography variant="h5" gutterBottom>Аналитика по курьерам</Typography>
              <CourierStatsTable couriers={courierStats} />
            </Paper>
          } />
        </Routes>
      </Container>
    </Router>
  );
}
