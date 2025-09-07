// controllers/localAuthController.js
import { auth } from "../config/lucia.js";
import pool from "../config/db.js";
import { Argon2id } from "oslo/password";
import { randomBytes } from "crypto";

/**
 * Util: génère un id aléatoire hex (30 chars par défaut)
 */
function genId(n = 15) {
  return randomBytes(n).toString("hex");
}

/**
 * POST /api/auth/signup
 */
export const signup = async (req, res) => {
  try {
    const { name, email, password } = req.body ?? {};
    if (!name || !email || !password) {
      return res
        .status(400)
        .json({ message: "Champs requis: name, email, password" });
    }

    const lowerEmail = String(email).toLowerCase().trim();

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

    // Crée la session Lucia et envoie le cookie via les helpers
    const session = await auth.createSession({
      userId,
      attributes: {},
    });

    const cookie = auth.createSessionCookie(session.id);
    res.append("Set-Cookie", cookie.serialize());

    return res.status(201).json({ message: "Utilisateur créé avec succès" });
  } catch (err) {
    console.error("SIGNUP ERROR:", err);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};

/**
 * POST /api/auth/login
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

    // Crée la session et envoie le cookie via les helpers Lucia
    const session = await auth.createSession({
      userId: u.id,
      attributes: {},
    });

    const cookie = auth.createSessionCookie(session.id);
    res.append("Set-Cookie", cookie.serialize());

    return res.json({ message: "Connexion réussie" });
  } catch (err) {
    console.error("LOGIN ERROR:", err);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};

/**
 * POST /api/auth/logout
 */
export const logout = async (req, res) => {
  try {
    // Récupère l'id de session depuis le cookie
    const raw = req.headers.cookie || "";
    const found = raw
      .split(";")
      .map((c) => c.trim())
      .find((c) => c.startsWith("auth_session="));
    const sessionId = found ? decodeURIComponent(found.split("=", 2)[1]) : null;

    if (sessionId) {
      try {
        await auth.invalidateSession(sessionId);
      } catch (_) {
        // session déjà invalide: on ignore
      }
    }

    // Envoie un cookie "blank" via Lucia pour purger côté client
    const blank = auth.createBlankSessionCookie();
    res.append("Set-Cookie", blank.serialize());

    return res.json({ message: "Déconnecté" });
  } catch (err) {
    console.error("LOGOUT ERROR:", err);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};
