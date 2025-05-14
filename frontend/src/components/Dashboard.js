// src/components/Dashboard.js

import React, { useState, useEffect } from "react";
import API_URLS from "../config";
import CourierMap from "./CourierMap";
import { HashRouter as Router, Routes, Route, NavLink, Navigate } from "react-router-dom";
import OrdersTable from "./OrdersTable";
import CourierStatsTable from "./CourierStatsTable";

export default function Dashboard({ token, onLogout }) {
  const [orders, setOrders] = useState([]);
  const [couriers, setCouriers] = useState([]);
  const [sel, setSel] = useState("");
  const [newCourierName, setNewCourierName] = useState("");
  const [newCourierPhone, setNewCourierPhone] = useState("");
  const [newCourierVehicle, setNewCourierVehicle] = useState("foot");
  const [newCourierLat, setNewCourierLat] = useState(55.7558);
  const [newCourierLng, setNewCourierLng] = useState(37.6176);
  const [courierStats, setCourierStats] = useState([]);
  const [newOrder, setNewOrder] = useState({
    senderName: "",
    recipientName: "",
    addressFrom: "",
    addressTo: "",
    weight: "",
    length: "",
    width: "",
    height: "",
    urgency: "1",
  });

  // Загрузка курьеров и заказов
  useEffect(() => {
    // курьеры
    fetch(`${API_URLS.orders}/couriers`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(res => {
        if (!res.ok) throw new Error("Не удалось загрузить курьеров");
        return res.json();
      })
      .then(data => {
        setCouriers(Array.isArray(data) ? data : []);
      })
      .catch(err => {
        console.error(err);
        setCouriers([]);
      });

    // заказы
    fetch(`${API_URLS.orders}/orders`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(async res => {
        const text = await res.text();
        try {
          return JSON.parse(text);
        } catch (e) {
          console.error("Ответ /orders не JSON:", text);
          throw new Error("Сервер вернул некорректный ответ");
        }
      })
      .then(data => {
        const list = Array.isArray(data) ? data : [];
        setOrders(list);
        if (list.length) setSel(list[0].id);
      })
      .catch(err => {
        console.error(err);
        setOrders([]);
      });
  }, [token]);

  useEffect(() => {
  fetch(`${API_URLS.analytics}/analytics/couriers`, {
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(res => {
      if (!res.ok) throw new Error("Не удалось загрузить статистику курьеров");
      return res.json();
    })
    .then(data => {
      setCourierStats(Array.isArray(data) ? data : []);
    })
    .catch(err => {
      console.error(err);
      setCourierStats([]);
    });
  }, [token]);

  // Обработчики
  const handleInputChange = e => {
    const { name, value } = e.target;
    setNewOrder(prev => ({ ...prev, [name]: value }));
  };

  const handleAddOrder = e => {
    e.preventDefault();
    const payload = {
      sender_name: newOrder.senderName,
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
      .then(res => {
        if (!res.ok) throw new Error("Не удалось добавить заказ");
        return res.json();
      })
      .then(o => {
        setOrders(prev => [...prev, o]);
        setSel(o.id);
        setNewOrder({ senderName: "", recipientName: "", addressFrom: "", addressTo: "", weight: "", length: "", width: "", height: "", urgency: "1" });
      })
      .catch(console.error);
  };

  const handleAddCourier = e => {
    e.preventDefault();
    fetch(`${API_URLS.orders}/couriers`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        name: newCourierName,
        phone: newCourierPhone,
        vehicle_type: newCourierVehicle,
        latitude: parseFloat(newCourierLat),
        longitude: parseFloat(newCourierLng),
        status: "available",
        active_order_id: "",
      }),
    })
      .then(res => {
        if (!res.ok) throw new Error("Не удалось добавить курьера");
        return res.json();
      })
      .then(c => {
        setCouriers(prev => [...prev, c]);
        setNewCourierName("");
        setNewCourierPhone("");
        setNewCourierVehicle("foot");
        setNewCourierLat(55.7558);
        setNewCourierLng(37.6176);
      })
      .catch(console.error);
  };

  const handleDeleteOrder = id => {
    fetch(`${API_URLS.orders}/orders/${id}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(res => {
        if (!res.ok) throw new Error();
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
      .then(res => {
        if (!res.ok) throw new Error("Не удалось назначить курьера");
        return res.json();
      })
      .then(() => {
        setOrders(prev => prev.map(o =>
          o.id === orderId
            ? { ...o, courier_id: courierId }
            : o
        ));
        // перезагружаем карту при новом курьере
        if (sel === orderId) setSel(orderId);
      })
      .catch(console.error);
  };

  return (
    <Router>
      <nav style={{ marginBottom: 16 }}>
        <NavLink to="/add-order" style={{ marginRight: 8 }}>Добавить заказ</NavLink>
        <NavLink to="/add-courier" style={{ marginRight: 8 }}>Добавить курьера</NavLink>
        <NavLink to="/orders" style={{ marginRight: 8 }}>Список заказов</NavLink>
        <NavLink to="/tracking" style={{ marginRight: 8 }}>Отслеживание</NavLink>
        <NavLink to="/couriers-stats" style={{ marginRight: 8 }}>Аналитика по курьерам</NavLink>

        <button onClick={onLogout} style={{ float: "right" }}>Выйти</button>
      </nav>
      <Routes>
        <Route path="/" element={<Navigate to="/add-order" replace />} />
        <Route
          path="/add-order"
          element={
            <form onSubmit={handleAddOrder}>
              <h3>Добавить заказ</h3>
              <div>
                <label>
                  Имя отправителя:
                  <input
                    type="text"
                    name="senderName"
                    value={newOrder.senderName}
                    onChange={handleInputChange}
                    required
                  />
                </label>
              </div>
              <div>
                <label>
                  Имя получателя:
                  <input
                    type="text"
                    name="recipientName"
                    value={newOrder.recipientName}
                    onChange={handleInputChange}
                    required
                  />
                </label>
              </div>
              <div>
                <label>
                  Адрес отправления:
                  <input
                    type="text"
                    name="addressFrom"
                    value={newOrder.addressFrom}
                    onChange={handleInputChange}
                    required
                  />
                </label>
              </div>
              <div>
                <label>
                  Адрес доставки:
                  <input
                    type="text"
                    name="addressTo"
                    value={newOrder.addressTo}
                    onChange={handleInputChange}
                    required
                  />
                </label>
              </div>
              <div>
                <label>
                  Вес (кг):
                  <input
                    type="number"
                    name="weight"
                    step="0.01"
                    value={newOrder.weight}
                    onChange={handleInputChange}
                    required
                  />
                </label>
              </div>
              <div>
                <label>
                  Длина (м):
                  <input
                    type="number"
                    name="length"
                    step="0.01"
                    value={newOrder.length}
                    onChange={handleInputChange}
                    required
                  />
                </label>
              </div>
              <div>
                <label>
                  Ширина (м):
                  <input
                    type="number"
                    name="width"
                    step="0.01"
                    value={newOrder.width}
                    onChange={handleInputChange}
                    required
                  />
                </label>
              </div>
              <div>
                <label>
                  Высота (м):
                  <input
                    type="number"
                    name="height"
                    step="0.01"
                    value={newOrder.height}
                    onChange={handleInputChange}
                    required
                  />
                </label>
              </div>
              <div>
                <label>
                  Срочность:
                  <select
                    name="urgency"
                    value={newOrder.urgency}
                    onChange={handleInputChange}
                  >
                    <option value="1">Стандартная</option>
                    <option value="2">Экспресс</option>
                  </select>
                </label>
              </div>
              <button type="submit" style={{ marginTop: "1rem" }}>
                Добавить
              </button>
            </form>
          }
        />
        <Route
          path="/add-courier"
          element={
            <form onSubmit={handleAddCourier}>
              <h3>Добавить курьера</h3>
              <div>
                <label>
                  Имя:
                  <input
                    type="text"
                    placeholder="Имя курьера"
                    value={newCourierName}
                    onChange={e => setNewCourierName(e.target.value)}
                    required
                  />
                </label>
              </div>
              <div>
                <label>
                  Телефон:
                  <input
                    type="text"
                    placeholder="+7..."
                    value={newCourierPhone}
                    onChange={e => setNewCourierPhone(e.target.value)}
                  />
                </label>
              </div>
              <div>
                <label>
                  Тип транспорта:
                  <select
                    value={newCourierVehicle}
                    onChange={e => setNewCourierVehicle(e.target.value)}
                  >
                    <option value="foot">Пеший</option>
                    <option value="bike">Велосипед</option>
                    <option value="car">Автомобиль</option>
                  </select>
                </label>
              </div>
              <div>
                <label>
                  Начальная широта:
                  <input
                    type="number"
                    step="0.0001"
                    value={newCourierLat}
                    onChange={e => setNewCourierLat(e.target.value)}
                  />
                </label>
              </div>
              <div>
                <label>
                  Начальная долгота:
                  <input
                    type="number"
                    step="0.0001"
                    value={newCourierLng}
                    onChange={e => setNewCourierLng(e.target.value)}
                  />
                </label>
              </div>
              <button type="submit" style={{ marginTop: "1rem" }}>Добавить курьера</button>
            </form>
          }
        />
        <Route
          path="/orders"
          element={
            <>
              <h3>Список заказов</h3>
              <OrdersTable
                orders={orders}couriers={couriers}
                onDeleteOrder={handleDeleteOrder}
                onAssignCourier={handleAssignCourier}
                ></OrdersTable>
              </>
          }
        />
        <Route
          path="/tracking"
          element={
            <>
              <h3>Отслеживание</h3>
              {orders.length ? (
                <select value={sel} onChange={e => setSel(e.target.value)}>
                  {orders.map(o => (
                    <option key={o.id} value={o.id}>
                      {o.sender_name} → {o.recipient_name}
                    </option>
                  ))}
                </select>
              ) : (
                <p>Нет заказов</p>
              )}
              {sel && (
                <CourierMap orderId={sel} token={token} />
              )}
            </>
          }
        />
        <Route
          path="/couriers-stats"
          element={
            <>
              <h3>Аналитика по курьерам</h3>
              <CourierStatsTable couriers={courierStats} />
            </>
          }
        />
      </Routes>
    </Router>
  );
}
