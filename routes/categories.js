// routes/categories.js
import express from "express";
import pool from "../config/db.js"; 

const router = express.Router();

// ✅ Liste des catégories principales
router.get("/categories", async (req, res) => {
  try {
    const [rows] = await pool.execute("SELECT id, name FROM categories");
    res.json(rows);
  } catch (err) {
    console.error(err);
    res.status(500).json({ error: "Erreur lors de la récupération des catégories" });
  }
});

// ✅ Liste des sous-catégories en fonction de la catégorie
router.get("/subcategories", async (req, res) => {
  const { category_id } = req.query;
  if (!category_id) return res.status(400).json({ error: "category_id requis" });

  try {
    const [rows] = await pool.execute(
      "SELECT id, name FROM subcategories WHERE category_id = ?",
      [category_id]
    );
    res.json(rows);
  } catch (err) {
    console.error(err);
    res.status(500).json({ error: "Erreur lors de la récupération des sous-catégories" });
  }
});

export default router;
