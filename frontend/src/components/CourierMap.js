import React, { useState, useEffect } from 'react';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet';
import L from 'leaflet';
import API_URLS from '../config';
import 'leaflet/dist/leaflet.css';

// Иконки Leaflet
delete L.Icon.Default.prototype._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: require('leaflet/dist/images/marker-icon-2x.png'),
  iconUrl:       require('leaflet/dist/images/marker-icon.png'),
  shadowUrl:     require('leaflet/dist/images/marker-shadow.png'),
});

export default function CourierMap({ orderId, token }) {
  const [orderData,  setOrderData]  = useState(null);
  const [fromMarker, setFromMarker] = useState(null);
  const [toMarker,   setToMarker]   = useState(null);
  const [courierPos, setCourierPos] = useState(null);

  // 1) Получаем детали заказа (содержит courier_id)
  useEffect(() => {
    if (!orderId) return;
    fetch(`${API_URLS.orders}/orders/${orderId}`, {
      headers: { Authorization: `Bearer ${token}` }
    })
      .then(res => res.ok ? res.json() : Promise.reject())
      .then(setOrderData)
      .catch(() => setOrderData(null));
  }, [orderId, token]);

  // 2) Геокодим адреса отправителя/получателя
  useEffect(() => {
    if (!orderData) return;
    const geocode = async (addr, setter) => {
      try {
        const res = await fetch(
          `https://nominatim.openstreetmap.org/search?format=json&q=${encodeURIComponent(addr)}`
        );
        const arr = await res.json();
        if (arr.length) {
          setter({
            position: [parseFloat(arr[0].lat), parseFloat(arr[0].lon)],
            label:    arr[0].display_name,
          });
        }
      } catch {
        setter(null);
      }
    };
    geocode(orderData.address_from, setFromMarker);
    geocode(orderData.address_to,   setToMarker);
  }, [orderData]);

  // 3) Поллинг за позицией курьера по его ID
  useEffect(() => {
    if (!orderData || !orderData.courier_id) return;
    const courierId = orderData.courier_id;
    const endpoint  = `${API_URLS.tracking}/couriers/tracking/${courierId}`;

    const tick = async () => {
      try {
        const res = await fetch(endpoint, {
          headers: { Authorization: `Bearer ${token}` }
        });
        if (res.ok) {
          // Поддерживаем поля latitude и longtitude, а также longitude
          const data = await res.json();
          const latitude = data.latitude;
          const longitude = data.longitude ?? data.longtitude;
          if (latitude != null && longitude != null) {
            setCourierPos([latitude, longitude]);
          }
        }
      } catch {
        // игнорируем ошибки
      }
    };

    // Сразу и потом каждые 5 секунд
    tick();
    const id = setInterval(tick, 5000);
    return () => clearInterval(id);
  }, [orderData?.courier_id, token]);

  // Центр карты: либо курьер, либо Москва
  const center = courierPos || [55.7558, 37.6176];
  useEffect(() => {
    console.log("✅ courierPos:", courierPos);
  }, [courierPos]);
  useEffect(() => {
  console.log("🧾 orderData:", orderData);
}, [orderData]);

const flag = document.querySelector('.leaflet-attribution-flag');
if (flag) {
  flag.remove();
}
  return (
    <MapContainer center={center} zoom={12} style={{ height: '400px', width: '100%' }}>
      <TileLayer
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        attribution="&copy; OpenStreetMap contributors"
      />

      {fromMarker && (
        <Marker position={fromMarker.position}>
          <Popup>Отправитель:<br />{fromMarker.label}</Popup>
        </Marker>
      )}
      {toMarker && (
        <Marker position={toMarker.position}>
          <Popup>Получатель:<br />{toMarker.label}</Popup>
        </Marker>
      )}
      {courierPos && (
        <Marker position={courierPos}>
          <Popup>Курьер здесь.<br />Order ID: {orderId}</Popup>
        </Marker>
      )}
    </MapContainer>
  );
}
