// controllers/authcontroller.js
import bcrypt from "bcryptjs";
import jwt from "jsonwebtoken";
import pool from "../config/db.js";

/** Génère un JWT.
 *  On peut garder un companyId *dérivé* (nullable) dans le token pour l’app.
 */
function generateToken(user) {
  return jwt.sign(
    {
      id: user.id,
      email: user.email,
      companyId: user.companyId ?? null, // dérivé de company_users
    },
    process.env.JWT_SECRET,
    { expiresIn: "7d" }
  );
}

/** Récupère les métadonnées société (companyId choisi, isCompanyAdmin, companyName) via company_users */
async function fetchCompanyMetaForUser(userId) {
  // isCompanyAdmin + companyId "principal" (priorité admin)
  const [meta] = await pool.query(
    `
    SELECT
      EXISTS(
        SELECT 1 FROM company_users cu
        WHERE cu.userId = ? AND LOWER(TRIM(cu.role)) = 'admin'
      ) AS isCompanyAdmin,
      (
        SELECT cu.companyId
        FROM company_users cu
        WHERE cu.userId = ?
        ORDER BY (LOWER(TRIM(cu.role)) = 'admin') DESC, cu.companyId ASC
        LIMIT 1
      ) AS companyId
    `,
    [userId, userId]
  );
  const m = meta[0] || {};
  let companyName = null;
  if (m.companyId) {
    const [c] = await pool.query("SELECT name FROM companies WHERE id = ? LIMIT 1", [m.companyId]);
    companyName = c[0]?.name ?? null;
  }
  return {
    companyId: m.companyId ?? null,
    companyName,
    isCompanyAdmin: !!m.isCompanyAdmin,
  };
}

/* =============================
   REGISTER
   ============================= */
export const register = async (req, res) => {
  try {
    const { email, password, name } = req.body || {};
    if (!email || !password) {
      return res.status(400).json({ message: "Email et mot de passe requis" });
    }

    const normEmail = String(email).trim().toLowerCase();

    const [existing] = await pool.query(
      "SELECT id FROM users WHERE LOWER(email) = ? LIMIT 1",
      [normEmail]
    );
    if (existing.length > 0) {
      return res.status(400).json({ message: "Cet utilisateur existe déjà" });
    }

    const hashedPassword = await bcrypt.hash(password, 10);
    const displayName = (name && String(name).trim()) || normEmail.split("@")[0];

    const [ins] = await pool.query(
      `INSERT INTO users (email, passwordHash, name, provider, is_admin)
       VALUES (?, ?, ?, 'local', 0)`,
      [normEmail, hashedPassword, displayName]
    );

    const userId = ins.insertId;
    const meta = await fetchCompanyMetaForUser(userId);

    const payloadUser = {
      id: userId,
      email: normEmail,
      name: displayName,
      isAdmin: false,
      companyId: meta.companyId,
      companyName: meta.companyName,
      isCompanyAdmin: meta.isCompanyAdmin,
    };

    const token = generateToken(payloadUser);
    return res.status(201).json({ user: payloadUser, token });
  } catch (err) {
    console.error("REGISTER ERROR:", err);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};

/* =============================
   LOGIN (solution 2 : via company_users)
   ============================= */
export const login = async (req, res) => {
  try {
    const { email, password } = req.body || {};
    if (!email || !password) {
      return res.status(400).json({ message: "Email et mot de passe requis" });
    }

    const [rows] = await pool.query(
      `
      SELECT
        u.id, u.email, u.name,
        u.is_admin AS isAdmin,
        u.passwordHash, u.provider
      FROM users u
      WHERE LOWER(u.email) = LOWER(?)
      LIMIT 1
      `,
      [email]
    );
    if (rows.length === 0) {
      return res.status(401).json({ message: "Identifiants invalides" });
    }

    const u = rows[0];

    if (!u.passwordHash) {
      return res.status(400).json({
        message:
          "Ce compte utilise la connexion sociale. Réinitialisez le mot de passe pour activer la connexion par e-mail.",
      });
    }

    const ok = await bcrypt.compare(password, u.passwordHash);
    if (!ok) return res.status(401).json({ message: "Identifiants invalides" });

    const meta = await fetchCompanyMetaForUser(u.id);

    const payloadUser = {
      id: u.id,
      email: u.email,
      name: u.name,
      isAdmin: !!u.isAdmin,
      companyId: meta.companyId,
      companyName: meta.companyName,
      isCompanyAdmin: meta.isCompanyAdmin,
    };

    const token = generateToken(payloadUser);
    return res.json({ user: payloadUser, token });
  } catch (err) {
    console.error("LOGIN ERROR:", err);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};

/* =============================
   SOCIAL LOGIN (Google/Facebook/Apple)
   ============================= */
export const social = async (req, res) => {
  try {
    const { email, name, provider } = req.body || {};
    if (!email) return res.status(400).json({ message: "Email requis" });

    const normEmail = String(email).trim().toLowerCase();

    let userId = null;
    const [rows] = await pool.query(
      "SELECT id, email, name, is_admin AS isAdmin FROM users WHERE LOWER(email) = ? LIMIT 1",
      [normEmail]
    );

    if (rows.length === 0) {
      // créer un utilisateur "social"
      const displayName = (name && String(name).trim()) || normEmail.split("@")[0];
      const [ins] = await pool.query(
        `INSERT INTO users (email, name, provider, is_admin)
         VALUES (?, ?, ?, 0)`,
        [normEmail, displayName, provider || null]
      );
      userId = ins.insertId;
    } else {
      userId = rows[0].id;
    }

    const [uRows] = await pool.query(
      "SELECT id, email, name, is_admin AS isAdmin FROM users WHERE id = ? LIMIT 1",
      [userId]
    );
    const u = uRows[0];

    const meta = await fetchCompanyMetaForUser(u.id);

    const payloadUser = {
      id: u.id,
      email: u.email,
      name: u.name,
      isAdmin: !!u.isAdmin,
      companyId: meta.companyId,
      companyName: meta.companyName,
      isCompanyAdmin: meta.isCompanyAdmin,
    };

    const token = generateToken(payloadUser);
    return res.json({ user: payloadUser, token });
  } catch (err) {
    console.error("SOCIAL LOGIN ERROR:", err);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};

/* =============================
   ME (profil connecté)
   ============================= */
export const me = async (req, res) => {
  try {
    const [rows] = await pool.query(
      `
      SELECT id, email, name, is_admin AS isAdmin
      FROM users
      WHERE id = ?
      LIMIT 1
      `,
      [req.user.id]
    );

    if (rows.length === 0) {
      return res.status(404).json({ message: "Utilisateur introuvable" });
    }

    const u = rows[0];
    const meta = await fetchCompanyMetaForUser(u.id);

    return res.json({
      id: u.id,
      email: u.email,
      name: u.name,
      isAdmin: !!u.isAdmin,
      companyId: meta.companyId,
      companyName: meta.companyName,
      isCompanyAdmin: meta.isCompanyAdmin,
    });
  } catch (err) {
    console.error("ME ERROR:", err);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};
