// src/components/Dashboard.js

import React, { useState, useEffect } from "react";
import API_URLS from "../config";
import CourierMap from "./CourierMap";

export default function Dashboard({ token, onLogout }) {
  const [orders, setOrders] = useState([]);
  const [couriers, setCouriers] = useState([]);
  const [sel, setSel] = useState("");
  const [newCourierName, setNewCourierName] = useState("");
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
  useEffect(() => {
    fetch(`${API_URLS.orders}/couriers`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(res => {
        if (!res.ok) throw new Error('Не удалось загрузить курьеров');
        return res.json();
      })
      .then(data => setCouriers(data))
      .catch(console.error);
  }, [token]);
  useEffect(() => {
    fetch(`${API_URLS.orders}/orders`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((r) => r.json())
      .then((data) => {
        if (Array.isArray(data) && data.length) {
          setOrders(data);
          setSel(data[0].id.toString());
        } else {
          setOrders([]); // сбрасываем список
        }
      })
      .catch(console.error);
  }, [token]);
  const handleAddCourier = (e) => {
    e.preventDefault();
    fetch(`${API_URLS.orders}/couriers`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ name: newCourierName }),
    })
      .then((res) => {
        if (!res.ok) throw new Error("Не удалось добавить курьера");
        return res.json();
      })
      .then((courier) => {
        // Обновляем локальный список курьеров:
        setCouriers((prev) => [...prev, courier]);
        // Очищаем поле ввода:
        setNewCourierName("");
      })
      .catch(console.error);
  };
  // 3) Функция назначения курьера на заказ
  const handleAssignCourier = (orderId, courierId) => {
    fetch(`${API_URLS.orders}/orders/${orderId}/assign-courier`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ courier_id: courierId }),
    })
      .then(res => {
        if (!res.ok) throw new Error('Ошибка при назначении курьера');
        return res.json();
      })
      .then(() => {
        // Обновляем локальный стейт: присваиваем courier_id нужному заказу
        setOrders(prev =>
          prev.map(o =>
            o.id === orderId
              ? { ...o, courier_id: courierId }
              : o
          )
        );
      })
      .catch(console.error);
  };
  const handleDeleteOrder = (id) => {
    fetch(`${API_URLS.orders}/orders/${id}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((r) => {
        if (!r.ok) throw new Error("Ошибка при удалении заказа");
        setOrders((prev) => prev.filter((o) => o.id !== id));
        if (sel === id.toString()) {
          const remaining = orders.filter((o) => o.id !== id);
          setSel(remaining.length ? remaining[0].id.toString() : "");
        }
      })
      .catch(console.error);
  };

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setNewOrder((prev) => ({ ...prev, [name]: value }));
  };

  const handleAddOrder = (e) => {
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
      .then((r) => {
        if (!r.ok) throw new Error("Ошибка при создании заказа");
        return r.json();
      })
      .then((data) => {
        setOrders((prev) => [...prev, data]);
        setSel(data.id);
        setNewOrder({
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
      })
      .catch(console.error);
  };

  return (
    <div>
      <h3>Добавить заказ</h3>
      <form onSubmit={handleAddOrder}>
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
              step="0.01"
              name="weight"
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
              step="0.01"
              name="length"
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
              step="0.01"
              name="width"
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
              step="0.01"
              name="height"
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
        <button type="submit">Добавить</button>
      </form>
      <h3>Добавить курьера</h3>
      <form onSubmit={handleAddCourier} style={{ marginBottom: "1rem" }}>
        <input
          type="text"
          placeholder="Имя курьера"
          value={newCourierName}
          onChange={e => setNewCourierName(e.target.value)}
          required
        />
        <button type="submit" style={{ marginLeft: "0.5rem" }}>
          Добавить курьера
        </button>
      </form>
      <h3>Список заказов</h3>
      {Array.isArray(orders) && orders.length ? (
        <ul>
          {orders.map((order) => (
            <li key={order.id} style={{ marginBottom: "1rem" }}>
              {order.sender_name} → {order.recipient_name} (Статус: {order.status})
              <button
                style={{ marginLeft: "1rem" }}
                onClick={() => handleDeleteOrder(order.id)}
              >
                Удалить
              </button>
              <div style={{ marginTop: "0.5rem" }}>
                <strong>Курьер:</strong>{" "}
                {order.courier_id
                  ? (couriers.find(c => c.id === order.courier_id)?.name || "Не найден")
                  : "Не назначен"}
                <select
                  value={order.courier_id || ""}
                  onChange={e => handleAssignCourier(order.id, e.target.value)}
                  style={{ marginLeft: "0.5rem" }}
                >
                  <option value="">— выбрать курьера —</option>
                  {couriers.map(c => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>
              </div>
            </li>
          ))}
        </ul>
      ) : (
        <p>Нет заказов</p>
      )}

      <h3>Выберите заказ для отслеживания</h3>
      {orders.length ? (
        <select value={sel} onChange={(e) => setSel(e.target.value)}>
          {orders.map((o) => (
            <option key={o.id} value={o.id}>
              {o.sender_name} → {o.recipient_name} (#{o.id})
            </option>
          ))}
        </select>
      ) : (
        <p>Нет заказов</p>
      )}
      {sel && <CourierMap orderId={sel} />}

      <br />
      <button onClick={onLogout}>Выйти</button>
    </div>
  );
}
