// server.js
import express from "express";
import cors from "cors";
import dotenv from "dotenv";

import authRoutes from "./routes/auth.js";
import productRoutes from "./routes/products.js";
import categoryRoutes from "./routes/categories.js";
import checkoutRoutes from "./routes/checkout.js";
import addressesRoutes from "./routes/addresses.js";
import companyRoutes from "./routes/company.js";

dotenv.config();

const app = express();
app.use(cors());
app.use(express.json());

app.get("/", (_req, res) => res.json({ status: "ok" }));

// Auth
app.use("/api/auth", authRoutes);

// Produits / Uploads
app.use("/api/products", productRoutes);
app.use("/api/uploads", express.static("uploads"));

// Catégories
app.use("/api/categories", categoryRoutes);

// Adresses
app.use("/api/addresses", addressesRoutes);

app.use("/api/company", companyRoutes);

// Paiement
app.use("/api/checkout", checkoutRoutes);

const PORT = process.env.PORT || 5000;
app.listen(PORT, "0.0.0.0", () => {
  console.log(`🚀 API démarrée sur http://0.0.0.0:${PORT}`);
});
