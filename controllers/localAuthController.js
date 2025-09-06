// controllers/localAuthController.js
import { auth } from "../config/lucia.js";
import pool from "../config/db.js";
import { Argon2id } from "oslo/password";
import { randomBytes } from "crypto";

/**
 * Util: génère un id texte pour users.id
 */
function genId(n = 15) {
  return randomBytes(n).toString("hex"); // 30 chars
}

/**
 * POST /api/auth/signup
 * body: { name, email, password }
 * -> insère dans users (hashed_password, flags) puis crée une session Lucia
 */
export const signup = async (req, res) => {
  try {
    const { name, email, password } = req.body ?? {};
    if (!name || !email || !password) {
      return res.status(400).json({ message: "Champs requis: name, email, password" });
    }

    const lowerEmail = String(email).toLowerCase().trim();

    // Vérifie si email déjà utilisé
    const exists = await pool.query(
      `SELECT 1 FROM users WHERE email = $1 LIMIT 1`,
      [lowerEmail]
    );
    if (exists.rowCount > 0) {
      return res.status(409).json({ message: "Email déjà utilisé" });
    }

    const hashed = await new Argon2id().hash(password);
    const userId = genId();

    await pool.query(
      `
      INSERT INTO users (id, email, name, hashed_password, is_admin, is_suspended, provider)
      VALUES ($1, $2, $3, $4, $5, $6, $7)
      `,
      [userId, lowerEmail, name.trim(), hashed, false, false, "local"]
    );

    // crée session Lucia (cookie HttpOnly)
    const session = await auth.createSession({
      userId,
      attributes: {}, // tu peux y mettre des flags si besoin
    });
    auth.handleRequest(req, res).setSession(session);

    return res.status(201).json({ message: "Utilisateur créé avec succès" });
  } catch (err) {
    console.error("SIGNUP ERROR:", err);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};

/**
 * POST /api/auth/login
 * body: { email, password }
 * -> lit user, vérifie hash, crée session
 */
export const login = async (req, res) => {
  try {
    const { email, password } = req.body ?? {};
    if (!email || !password) {
      return res.status(400).json({ message: "Champs requis: email, password" });
    }

    const lowerEmail = String(email).toLowerCase().trim();

    const r = await pool.query(
      `SELECT id, hashed_password, is_suspended FROM users WHERE email = $1 LIMIT 1`,
      [lowerEmail]
    );
    if (r.rowCount === 0) {
      return res.status(401).json({ message: "Identifiants invalides" });
    }
    const u = r.rows[0];
    if (u.is_suspended) {
      return res.status(403).json({ message: "Compte suspendu" });
    }

    const ok = await new Argon2id().verify(u.hashed_password, password);
    if (!ok) {
      return res.status(401).json({ message: "Identifiants invalides" });
    }

    const session = await auth.createSession({
      userId: u.id,
      attributes: {},
    });
    auth.handleRequest(req, res).setSession(session);

    return res.json({ message: "Connexion réussie" });
  } catch (err) {
    console.error("LOGIN ERROR:", err);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};

/**
 * POST /api/auth/logout
 * -> invalide la session et supprime le cookie
 */
export const logout = async (req, res) => {
  try {
    const request = auth.handleRequest(req, res);
    const session = await request.validate();
    if (session) {
      await auth.invalidateSession(session.sessionId);
      request.setSession(null);
    }
    return res.json({ message: "Déconnecté" });
  } catch (err) {
    console.error("LOGOUT ERROR:", err);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};
