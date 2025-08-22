// routes/auth.js
import express from "express";
import { register, login, social, me } from "../controllers/authcontroller.js";
import { authMiddleware } from "../middleware/authMiddleware.js"; // ← export nommé

const router = express.Router();

// Auth publiques
router.post("/register", register);
router.post("/login", login);
router.post("/social", social);

// Auth protégée
router.get("/me", authMiddleware, me);

export default router;
