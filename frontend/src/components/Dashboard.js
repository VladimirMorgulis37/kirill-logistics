// src/components/Dashboard.js
import React, {useState,useEffect} from "react";
import API_URLS from "../config";
import CourierMap from "./CourierMap";

export default function Dashboard({token,onLogout}) {
  const [orders, setOrders] = useState([]);
  const [sel, setSel] = useState("");

  useEffect(() => {
    fetch(`${API_URLS.orders}/orders`)
      .then(r=>r.json())
      .then(data => {
        setOrders(data);
        if(data.length) setSel(data[0].id);
      }).catch(console.error);
  }, []);

  return (
    <div>
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
      <h3>Выберите заказ для отслеживания</h3>
      {orders.length ? (
        <select value={sel} onChange={e=>setSel(e.target.value)}>
          {orders.map(o=>(
            <option key={o.id} value={o.id}>
              {o.sender_name} → {o.recipient_name} (#{o.id})
            </option>
          ))}
        </select>
      ) : <p>Нет заказов</p>}

      {sel && <CourierMap orderId={sel}/>}
      <br></br>
      <button onClick={onLogout}>Выйти</button>
    </div>
  );
}
