// controllers/authController.js
import { auth } from "../config/lucia.js";
import pool from "../config/db.js";

async function fetchCompanyMetaForUser(userId) {
  const result = await pool.query(
    `
    SELECT
      EXISTS(
        SELECT 1 FROM company_users cu
        WHERE cu."userId" = $1 AND LOWER(TRIM(cu.role)) = 'admin'
      ) AS "isCompanyAdmin",
      (
        SELECT cu."companyId"
        FROM company_users cu
        WHERE cu."userId" = $1
        ORDER BY (LOWER(TRIM(cu.role)) = 'admin') DESC, cu."companyId" ASC
        LIMIT 1
      ) AS "companyId"
    `,
    [userId]
  );

  const meta = result.rows[0] || {};
  let companyName = null;

  if (meta.companyId) {
    const company = await pool.query(
      "SELECT name FROM companies WHERE id = $1 LIMIT 1",
      [meta.companyId]
    );
    companyName = company.rows[0]?.name ?? null;
  }

  return {
    companyId: meta.companyId ?? null,
    companyName,
    isCompanyAdmin: !!meta.isCompanyAdmin,
  };
}

async function loadUserAndCompanyData(userId) {
  const r = await pool.query(
    `SELECT id, email, name, is_admin AS "isAdmin" FROM users WHERE id = $1 LIMIT 1`,
    [userId]
  );
  if (r.rows.length === 0) throw new Error("Utilisateur introuvable");
  const user = r.rows[0];
  const meta = await fetchCompanyMetaForUser(user.id);
  return { user, meta };
}

export const me = async (req, res) => {
  try {
    const userId = req.session.user.userId;
    const { user, meta } = await loadUserAndCompanyData(userId);
    return res.json({
      id: user.id,
      email: user.email,
      name: user.name,
      isAdmin: !!user.isAdmin,
      companyId: meta.companyId,
      companyName: meta.companyName,
      isCompanyAdmin: meta.isCompanyAdmin,
    });
  } catch (err) {
    console.error("ME ERROR:", err);
    return res
      .status(err.message === "Utilisateur introuvable" ? 404 : 500)
      .json({ message: err.message || "Erreur serveur" });
  }
};

export const refreshUserData = async (req, res) => {
  try {
    const userId = req.session.user.userId;
    const { user, meta } = await loadUserAndCompanyData(userId);

    // recrée une session avec payload à jour
    await auth.invalidateSession(req.session.sessionId);
    const newSession = await auth.createSession({
      userId: user.id,
      attributes: {
        userId: user.id,
        name: user.name,
        isAdmin: !!user.isAdmin,
        companyId: meta.companyId,
        companyName: meta.companyName,
        isCompanyAdmin: meta.isCompanyAdmin,
      },
    });
    auth.handleRequest(req, res).setSession(newSession);

    return res.json({ message: "Données utilisateur mises à jour" });
  } catch (err) {
    console.error("REFRESH ERROR:", err);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};

export const deleteAccount = async (req, res) => {
  try {
    const userId = req.session.user.userId;

    const r = await pool.query(
      "SELECT id FROM users WHERE id = $1 LIMIT 1",
      [userId]
    );

    if (r.rows.length > 0) {
      const uid = r.rows[0].id;
      await pool.query('DELETE FROM company_users WHERE "userId" = $1', [uid]);
      await pool.query("DELETE FROM users WHERE id = $1", [uid]);
    }

    await auth.invalidateSession(req.session.sessionId);
    return res.json({ message: "Compte supprimé avec succès" });
  } catch (err) {
    console.error("DELETE ACCOUNT ERROR:", err);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};
