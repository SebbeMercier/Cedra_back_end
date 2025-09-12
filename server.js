import express from "express";
import cors from "cors";
import cookieParser from "cookie-parser";
import dotenv from "dotenv";

import { auth } from "./config/auth.js";
import { createAuthRouter } from "better-auth";  // ✅ le bon import

import productsRoutes from "./routes/products.js";
import categoriesRoutes from "./routes/categories.js";
import addressesRoutes from "./routes/addresses.js";
import companyRoutes from "./routes/company.js";
import checkoutRoutes from "./routes/checkout.js";

dotenv.config();

const app = express();
const PORT = process.env.PORT || 5000;

app.use(cors({
  origin: process.env.WEBSITE_DOMAIN || "http://localhost:3001",
  credentials: true
}));
app.use(cookieParser());
app.use(express.json());

// ⚡ Brancher Better Auth
app.use("/api/auth", createAuthRouter(auth));

app.use("/api/products", productsRoutes);
app.use("/api/uploads", express.static("uploads"));
app.use("/api/categories", categoriesRoutes);
app.use("/api/addresses", addressesRoutes);
app.use("/api/company", companyRoutes);
app.use("/api/checkout", checkoutRoutes);

app.get("/", (_req, res) => res.json({ status: "ok" }));

app.use((err, req, res, next) => {
  console.error("Global error handler:", err);
  res.status(500).json({ message: "Erreur serveur interne" });
});

app.listen(PORT, "0.0.0.0", () => {
  console.log(`🚀 API CEDRA démarrée sur http://0.0.0.0:${PORT}`);
});
