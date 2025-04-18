// src/index.js
import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import 'leaflet/dist/leaflet.css';

// Material‑UI imports
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

const root = ReactDOM.createRoot(document.getElementById("root"));
root.render(
  <ThemeProvider theme={theme}>
    <CssBaseline />
    <Container maxWidth="lg" sx={{ py: 4 }}>
      <App />
    </Container>
  </ThemeProvider>
);
