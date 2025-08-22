// routes/products.js
import express from "express";
import multer from "multer";
import pool from "../config/db.js"; 

const router = express.Router();

// ajout produit Cedra
const storage = multer.diskStorage({
  destination: (req, file, cb) => cb(null, "uploads/"),
  filename: (req, file, cb) =>
    cb(null, Date.now() + "-" + file.originalname),
});
const upload = multer({ storage });

// Ajouter un produit
router.post("/add", upload.single("image"), async (req, res) => {
  const { name, price, subcategory_id, description, stock, category_id } = req.body;

  if (!name || !price || !subcategory_id || !category_id || !description || !stock) {
    return res.status(400).json({ error: "Tous les champs sont requis." });
  }

  let image_url = null;
  if (req.file) {
    const host = req.protocol + "://" + req.get("host");
    image_url = `${host}/uploads/${req.file.filename}`;
  }

  try {
    const [result] = await pool.execute(
      "INSERT INTO products (name, description, price, stock, image_url, category_id, subcategory_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
      [name, description, price, stock, image_url, category_id, subcategory_id]
    );
    res.json({ message: "Produit ajouté", id: result.insertId });
  } catch (err) {
    console.error(err);
    res.status(500).json({ error: "Erreur lors de l'ajout du produit" });
  }
});

// Recherche Produit 
router.get("/search", async (req, res) => {
  const { q } = req.query;

  try {
    const [rows] = await pool.execute(
      "SELECT id, name, CAST(price AS DECIMAL(10,2)) AS price, image_url FROM products WHERE name LIKE ?",
      [`%${q || ""}%`]
    );
    res.json(rows);
  } catch (err) {
    console.error(err);
    res.status(500).json({ error: "Erreur lors de la recherche" });
  }
});

// ✅ Liste de tous les produits
router.get("/", async (req, res) => {
  try {
    const [rows] = await pool.execute(
      "SELECT id, name, CAST(price AS DECIMAL(10,2)) AS price, image_url FROM products"
    );
    res.json(rows);
  } catch (err) {
    console.error(err);
    res.status(500).json({ error: "Erreur lors de la récupération des produits" });
  }
});

export default router;
