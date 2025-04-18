// src/App.js

import React, { useState } from "react";
import LoginForm from "./components/LoginForm";
import Dashboard from "./components/Dashboard";
import "./App.css";

// Material‑UI
import { ThemeProvider, createTheme } from "@mui/material/styles";
import CssBaseline from "@mui/material/CssBaseline";
import Container from "@mui/material/Container";

const theme = createTheme({
  palette: {
    primary: { main: "#1976d2" },
    background: { default: "#f4f6f8" },
  },
  components: {
    MuiButton: {
      styleOverrides: {
        root: { textTransform: "none" }
      }
    }
  }
});

function App() {
  const [token, setToken] = useState(() => localStorage.getItem("token") || "");

  const handleLogin = (newToken) => {
    localStorage.setItem("token", newToken);
    setToken(newToken);
  };

  const handleLogout = () => {
    localStorage.removeItem("token");
    setToken("");
  };

  return (
    <ThemeProvider theme={theme}>
      {/* Сброс базовых стилей */}
      <CssBaseline />

      {/* Ограничим ширину и добавим отступы */}
      <Container maxWidth="lg" sx={{ py: 4 }}>
        {token ? (
          <Dashboard token={token} onLogout={handleLogout} />
        ) : (
          <LoginForm onLogin={handleLogin} />
        )}
      </Container>
    </ThemeProvider>
  );
}

export default App;
