// middleware/companyAdminMiddleware.js
import { pool } from "../config/db.js";

export default async function companyAdminMiddleware(req, res, next) {
  try {
    const companyId = req.user.companyId;
    if (!companyId) {
      return res.status(403).json({ message: "Aucune société liée au compte." });
    }

    const [rows] = await pool.query(
      "SELECT role FROM company_users WHERE userId = ? AND companyId = ? LIMIT 1",
      [req.user.id, companyId]
    );

    if (!rows.length || rows[0].role !== "admin") {
      return res.status(403).json({ message: "Accès réservé aux admins de la société." });
    }

    next();
  } catch (err) {
    console.error("companyAdminMiddleware error:", err);
    res.status(500).json({ message: "Erreur serveur" });
  }
}
