import React, { useState, useEffect } from 'react';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';

// Исправление стандартных иконок (иногда требуется, чтобы метки отображались корректно)
delete L.Icon.Default.prototype._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: require('leaflet/dist/images/marker-icon-2x.png'),
  iconUrl: require('leaflet/dist/images/marker-icon.png'),
  shadowUrl: require('leaflet/dist/images/marker-shadow.png'),
});

// Компонент принимает orderId, чтобы фильтровать сообщения по конкретному заказу.
function CourierMap({ orderId }) {
  const [position, setPosition] = useState(null);

  useEffect(() => {
    // Открываем WebSocket соединение к Tracking Service.
    const ws = new WebSocket("ws://localhost:8083/tracking/ws");

    ws.onopen = () => {
      console.log("WebSocket connection established for tracking");
    };

    ws.onmessage = (event) => {
      // Предполагается, что сообщение представляет собой JSON с полями order_id, latitude, longitude.
      try {
        const data = JSON.parse(event.data);
        // Если сообщение относится к нужному заказу, обновляем позицию.
        if (data.order_id === orderId) {
          console.log("Получено обновление трекинга для заказа", orderId, data);
          setPosition([data.latitude, data.longitude]);
        }
      } catch (err) {
        console.error("Ошибка парсинга сообщения:", err);
      }
    };

    ws.onerror = (err) => {
      console.error("WebSocket error:", err);
    };

    // При закрытии компонента закрываем соединение
    return () => ws.close();
  }, [orderId]);

  // Задаем позицию по умолчанию (например, центр Москвы)
  const defaultPosition = [55.7558, 37.6176];

  return (
    <MapContainer
      center={position || defaultPosition}
      zoom={10}
      style={{ height: '400px', width: '100%' }}
    >
      <TileLayer
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
      />
      {position && (
        <Marker position={position}>
          <Popup>
            Курьер находится здесь.<br /> OrderID: {orderId}
          </Popup>
        </Marker>
      )}
    </MapContainer>
  );
}

export default CourierMap;
