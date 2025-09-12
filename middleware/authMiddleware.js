// middleware/authMiddleware.js
import { auth } from "../config/auth.js";
import { pool } from "../config/db.js";

/**
 * Récupère l'id de session depuis les cookies de la requête.
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
 * Supprime le cookie auth côté client (cookie "blanc")
 */
function purgeSessionCookie(res) {
  const blank = auth.createBlankSessionCookie();
  res.append("Set-Cookie", blank.serialize());
}

/**
 * Auth obligatoire : bloque si non connecté
 */
export const authMiddleware = async (req, res, next) => {
  try {
    const sid = getSessionIdFromReq(req);
    if (!sid) return deny(res, 401, "Non authentifié");

    const { session, user } = await auth.validateSession(sid);

    if (!session) {
      purgeSessionCookie(res);
      return deny(res, 401, "Session invalide");
    }

    if (session.fresh) {
      const cookie = auth.createSessionCookie(session.id);
      res.append("Set-Cookie", cookie.serialize());
    }

    // Injection dans req
    req.session = session;
    req.userId = session.userId;
    req.authUser = await getFullUser(session.userId);

    return next();
  } catch (e) {
    console.error("AUTH MIDDLEWARE ERROR:", e);
    purgeSessionCookie(res);
    return deny(res, 401, "Authentification requise");
  }
};

/**
 * Auth facultative : continue même si la session est absente ou invalide
 */
export const optionalAuth = async (req, res, next) => {
  try {
    const sid = getSessionIdFromReq(req);
    if (!sid) return next();

    const { session, user } = await auth.validateSession(sid);

    if (!session) {
      purgeSessionCookie(res);
      return next();
    }

    if (session.fresh) {
      const cookie = auth.createSessionCookie(session.id);
      res.append("Set-Cookie", cookie.serialize());
    }

    req.session = session;
    req.userId = session.userId;
    req.authUser = await getFullUser(session.userId);

    return next();
  } catch {
    return next(); // invité silencieux
  }
};

/**
 * Rafraîchit la session si elle expire bientôt (< 7 jours)
 */
export const refreshSessionIfNeeded = async (req, res, next) => {
  try {
    if (!req.session?.expiresAt) return next();

    const expMs = new Date(req.session.expiresAt).getTime();
    if (!Number.isFinite(expMs)) return next();

    const now = Date.now();
    const sevenDays = 7 * 24 * 60 * 60 * 1000;

    if (expMs - now > sevenDays) return next();

    const newSession = await auth.createSession({
      userId: req.session.userId,
      attributes: req.session.attributes || {},
    });

    try {
      await auth.invalidateSession(req.session.id);
    } catch (_) {}

    const cookie = auth.createSessionCookie(newSession.id);
    res.append("Set-Cookie", cookie.serialize());

    req.session = newSession;
    req.userId = newSession.userId;

    return next();
  } catch {
    return next(); // ne bloque jamais
  }
};

/**
 * Middleware admin global
 */
export const requireAdmin = async (req, res, next) => {
  if (!req.authUser) return deny(res, 401, "Non authentifié");
  if (!req.authUser.isAdmin) return deny(res, 403, "Accès admin requis");
  return next();
};

/**
 * Middleware admin global OU admin d'entreprise
 */
export const requireCompanyAdmin = async (req, res, next) => {
  if (!req.authUser) return deny(res, 401, "Non authentifié");

  if (req.authUser.isAdmin || req.authUser.isCompanyAdmin) {
    return next();
  }

  return deny(res, 403, "Accès administrateur d’entreprise requis");
};

/**
 * Charge les données complètes de l'utilisateur
 */
async function getFullUser(userId) {
  try {
    const r = await pool.query(
      `
      SELECT 
        u.id,
        u.email,
        u.name,
        u.is_admin AS "isAdmin",
        u.is_suspended AS "isSuspended",
        CASE 
          WHEN cu.role ILIKE 'admin' THEN TRUE
          ELSE FALSE
        END AS "isCompanyAdmin"
      FROM users u
      LEFT JOIN company_users cu ON cu."userId" = u.id
      WHERE u.id = $1
      LIMIT 1
      `,
      [userId]
    );

    if (r.rowCount === 0) return null;

    const u = r.rows[0];
    return {
      userId: u.id,
      email: u.email,
      name: u.name,
      isAdmin: !!u.isAdmin,
      isSuspended: !!u.isSuspended,
      isCompanyAdmin: !!u.isCompanyAdmin,
    };
  } catch (e) {
    console.error("GET FULL USER ERROR:", e);
    return null;
  }
}

/**
 * Standardise les réponses d'erreur
 */
function deny(res, status = 403, message = "Accès refusé") {
  return res.status(status).json({ error: "AUTH_ERROR", message });
}