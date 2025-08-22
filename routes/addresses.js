// routes/addresses.js
import express from "express";
import { authMiddleware } from "../middleware/authMiddleware.js";
import { listMine, createAddress } from "../controllers/addressescontroller.js";

const router = express.Router();

// Perso + entreprise (privées & partagées) de l'utilisateur courant
router.get("/mine", authMiddleware, listMine);

// Créer une adresse perso ou entreprise (privée/partagée)
router.post("/", authMiddleware, createAddress);

export default router;
