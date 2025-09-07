// routes/auth.js
import { Router } from "express";
import { signup, login, logout } from "../controllers/localAuthController.js";
import { me } from "../controllers/authController.js";
import {
  authMiddleware,
  refreshSessionIfNeeded,
} from "../middleware/authMiddleware.js";

const router = Router();

// --- routes publiques ---
router.post("/signup", signup);
router.post("/login", login);

// --- routes protégées ---
// ordre important: valider session d'abord, puis refresh si besoin
router.use(authMiddleware);
router.use(refreshSessionIfNeeded);

router.post("/logout", logout);
router.get("/me", me);

export default router;