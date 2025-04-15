// src/components/Dashboard.js
import React, { useState, useEffect } from "react";
import API_URLS from "../config";

const Dashboard = ({ token, onLogout }) => {
  const [protectedData, setProtectedData] = useState("");
  const [orders, setOrders] = useState([]);

  // Пример запроса к защищённому эндпоинту Auth Service.
  useEffect(() => {
    const fetchProtected = async () => {
      try {
        const res = await fetch(`${API_URLS.auth}/protected`, {
          headers: { Authorization: token },
        });
        if (res.ok) {
          const data = await res.json();
          setProtectedData(data.message);
        } else {
          setProtectedData("Ошибка доступа");
        }
      } catch (err) {
        setProtectedData("Ошибка соединения");
      }
    };
    fetchProtected();
  }, [token]);

  // Пример запроса списка заказов из Order Service.
  useEffect(() => {
    const fetchOrders = async () => {
      try {
        const res = await fetch(`${API_URLS.orders}/orders`);
        if (res.ok) {
          const data = await res.json();
          setOrders(data);
        } else {
          console.error("Ошибка получения заказов");
        }
      } catch (err) {
        console.error("Ошибка соединения:", err);
      }
    };
    fetchOrders();
  }, []);

  return (
    <div>
      <h2>Панель администратора</h2>
      <p>Защищённое сообщение: {protectedData}</p>
      <h3>Список заказов</h3>
      {orders.length ? (
        <ul>
          {orders.map((order) => (
            <li key={order.id}>
              {order.sender_name} → {order.recipient_name} (Статус: {order.status})
            </li>
          ))}
        </ul>
      ) : (
        <p>Нет заказов</p>
      )}
      <button onClick={onLogout}>Выйти</button>
    </div>
  );
};

export default Dashboard;
