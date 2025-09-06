import express from "express";
import { signup, login, logout } from "../controllers/localAuthController.js";
import { me, refreshUserData, deleteAccount } from "../controllers/authController.js";
import { authMiddleware, requireAdmin, requireCompanyAdmin, refreshSessionIfNeeded } from "../middleware/authMiddleware.js";


const router = express.Router();


// Public
router.post("/signup", signup);
router.post("/login", login);
router.post("/logout", logout);


// Protégé
router.get("/me", authMiddleware, refreshSessionIfNeeded, me);
router.post("/refresh", authMiddleware, refreshUserData);
router.delete("/account", authMiddleware, deleteAccount);


// Autorisations
router.get("/admin-only", authMiddleware, requireAdmin, (_req, res) => {
res.json({ message: "Accès autorisé pour les admins" });
});


router.get("/company-admin", authMiddleware, requireCompanyAdmin, (_req, res) => {
res.json({ message: "Accès autorisé pour les admins d'entreprise" });
});


export default router;

