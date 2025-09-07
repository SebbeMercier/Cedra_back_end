import pool from "../config/db.js";

export const me = async (req, res) => {
  const userId =
    req.session?.userId ??
    res.locals?.session?.userId ??
    req.userId ??
    null;

  if (!userId) {
    return res.status(401).json({ message: "Non authentifié" });
  }

  try {
    const r = await pool.query(
      `SELECT id, email, name, is_admin, is_suspended, provider
       FROM users WHERE id = $1 LIMIT 1`,
      [userId]
    );
    if (r.rowCount === 0) return res.status(404).json({ message: "Utilisateur introuvable" });
    return res.json({ user: r.rows[0] });
  } catch (err) {
    console.error("ME ERROR:", err);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};
