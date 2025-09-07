// middleware/authMiddleware.js
import { auth } from "../config/lucia.js";
import pool from "../config/db.js";

/**
 * Récupère l'id de session depuis les cookies de la requête.
 * (cookie-parser monté dans server.js)
 */
function getSessionIdFromReq(req) {
  if (req.cookies?.auth_session) return req.cookies.auth_session;

  const raw = req.headers.cookie || "";
  const found = raw
    .split(";")
    .map((c) => c.trim())
    .find((c) => c.startsWith("auth_session="));
  return found ? decodeURIComponent(found.split("=", 2)[1]) : null;
}

/**
 * Middleware d'auth OBLIGATOIRE.
 * - Lit le cookie "auth_session"
 * - Valide via Lucia
 * - Hydrate req.session et req.userId
 * - Gère la rotation du cookie quand session.fresh === true
 */
export const authMiddleware = async (req, res, next) => {
  try {
    const sid = getSessionIdFromReq(req);
    if (!sid) {
      req.session = null;
      req.userId = null;
      return res.status(401).json({ message: "Non authentifié" });
    }

    const { session, user } = await auth.validateSession(sid);

    if (!session) {
      // Purge côté client si la session est invalide
      const blank = auth.createBlankSessionCookie();
      res.append("Set-Cookie", blank.serialize());
      req.session = null;
      req.userId = null;
      return res.status(401).json({ message: "Session invalide" });
    }

    // Rotation recommandée par Lucia quand session.fresh === true
    if (session.fresh) {
      const cookie = auth.createSessionCookie(session.id);
      res.append("Set-Cookie", cookie.serialize());
    }

    req.session = session; // { id, userId, expiresAt, fresh, attributes? }
    req.userId = session.userId;
    req.authUser = user ?? null; // si tu veux l'exposer

    return next();
  } catch (e) {
    console.error("AUTH MIDDLEWARE ERROR:", e);
    const blank = auth.createBlankSessionCookie();
    res.append("Set-Cookie", blank.serialize());
    return res.status(401).json({ message: "Authentification requise" });
  }
};

/**
 * Middleware d'auth OPTIONNELLE.
 * - Pas de cookie: continue sans session
 * - Cookie invalide: purge et continue sans session
 */
export const optionalAuth = async (req, res, next) => {
  try {
    const sid = getSessionIdFromReq(req);
    if (!sid) return next();

    const { session, user } = await auth.validateSession(sid);

    if (!session) {
      const blank = auth.createBlankSessionCookie();
      res.append("Set-Cookie", blank.serialize());
      req.session = null;
      req.userId = null;
      return next();
    }

    if (session.fresh) {
      const cookie = auth.createSessionCookie(session.id);
      res.append("Set-Cookie", cookie.serialize());
    }

    req.session = session;
    req.userId = session.userId;
    req.authUser = user ?? null;

    return next();
  } catch {
    // on passe en invité en cas d'erreur
    return next();
  }
};

/**
 * Rafraîchit la session si elle expire bientôt (ex: < 7 jours).
 * - Crée une NOUVELLE session, invalide l'ancienne,
 *   remplace le cookie.
 *
 * Remarque: ceci est optionnel. Lucia gère déjà la rotation via session.fresh.
 * Garde-le si tu veux une "extension" proactive à J-7.
 */
export const refreshSessionIfNeeded = async (req, res, next) => {
  try {
    if (!req.session || !req.session.expiresAt) {
      return next();
    }

    const expMs = new Date(req.session.expiresAt).getTime();
    if (!Number.isFinite(expMs)) return next();

    const now = Date.now();
    const sevenDays = 7 * 24 * 60 * 60 * 1000;

    // Si > 7 jours restants, on ne rafraîchit pas
    if (expMs - now > sevenDays) {
      return next();
    }

    // Proactif: créer une nouvelle session
    const newSession = await auth.createSession({
      userId: req.session.userId,
      attributes: req.session.attributes || {},
    });

    // Invalide l'ancienne (best effort)
    try {
      await auth.invalidateSession(req.session.id);
    } catch (_) {}

    // Émet le nouveau cookie via les helpers Lucia
    const cookie = auth.createSessionCookie(newSession.id);
    res.append("Set-Cookie", cookie.serialize());

    // Remplace en mémoire
    req.session = newSession;
    req.userId = newSession.userId;

    return next();
  } catch {
    // en cas d’erreur de refresh, on ne bloque pas la requête
    return next();
  }
};

/**
 * Vérifie le flag admin depuis la base (users.is_admin)
 */
export const requireAdmin = async (req, res, next) => {
  try {
    if (!req.session) return res.status(401).json({ message: "Non authentifié" });

    const r = await pool.query(
      `SELECT is_admin AS "isAdmin" FROM users WHERE id = $1 LIMIT 1`,
      [req.session.userId]
    );
    if (r.rowCount === 0) return res.status(403).json({ message: "Accès refusé" });
    if (!r.rows[0].isAdmin) return res.status(403).json({ message: "Accès admin requis" });

    return next();
  } catch (e) {
    console.error("ADMIN CHECK ERROR:", e);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};

/**
 * Vérifie admin d’entreprise (company_users.role='admin') ou admin global.
 */
export const requireCompanyAdmin = async (req, res, next) => {
  try {
    if (!req.session) return res.status(401).json({ message: "Non authentifié" });

    const userId = req.session.userId;

    // admin global ?
    const g = await pool.query(
      `SELECT is_admin AS "isAdmin" FROM users WHERE id = $1 LIMIT 1`,
      [userId]
    );
    if (g.rows[0]?.isAdmin) return next();

    // admin d'entreprise ?
    const c = await pool.query(
      `
      SELECT 1
      FROM company_users
      WHERE "userId" = $1 AND LOWER(TRIM(role)) = 'admin'
      LIMIT 1
      `,
      [userId]
    );
    if (c.rowCount === 0) {
      return res
        .status(403)
        .json({ message: "Accès administrateur d’entreprise requis" });
    }

    return next();
  } catch (e) {
    console.error("COMPANY ADMIN CHECK ERROR:", e);
    return res.status(500).json({ message: "Erreur serveur" });
  }
};

/**
 * Récupère un "profil" rapide depuis la session (enrichi via DB)
 */
export const getUserFromSession = async (req) => {
  if (!req.session) return null;
  const userId = req.session.userId;

  const r = await pool.query(
    `SELECT id, email, name, is_admin AS "isAdmin", is_suspended AS "isSuspended"
     FROM users WHERE id = $1 LIMIT 1`,
    [userId]
  );
  if (r.rowCount === 0) return { userId };

  const u = r.rows[0];
  return {
    supertokensUserId: null, // legacy placeholder si tu réutilises d'anciens champs
    userId: u.id,
    email: u.email,
    name: u.name,
    isAdmin: !!u.isAdmin,
    isSuspended: !!u.isSuspended,
  };
};
