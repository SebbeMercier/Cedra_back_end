import { auth } from "../config/lucia.js";


export const authMiddleware = async (req, res, next) => {
try {
const request = auth.handleRequest(req, res);
const session = await request.validate();
if (!session) return res.status(401).json({ message: "Non authentifié" });
req.session = session;
req.user = session.user;
next();
} catch (err) {
console.error("AUTH MIDDLEWARE ERROR:", err);
return res.status(401).json({ message: "Session invalide" });
}
};


export const refreshSessionIfNeeded = async (req, res, next) => {
try {
if (!req.session) return next();
if (req.headers["x-refresh-session"] === "1") {
const { session } = req;
const attrs = session.attributes ?? {};
await auth.invalidateSession(session.sessionId);
const newSession = await auth.createSession({ userId: session.user.userId, attributes: attrs });
auth.handleRequest(req, res).setSession(newSession);
req.session = newSession;
}
next();
} catch (err) {
console.error("REFRESH SESSION ERROR:", err);
return res.status(500).json({ message: "Erreur lors du rafraîchissement de session" });
}
};


export const requireAdmin = (req, res, next) => {
try {
const isAdmin = !!(req?.user?.isAdmin ?? req?.session?.attributes?.isAdmin);
if (!isAdmin) return res.status(403).json({ message: "Accès refusé - droits administrateur requis" });
next();
} catch (err) {
console.error("ADMIN CHECK ERROR:", err);
return res.status(500).json({ message: "Erreur serveur" });
}
};


export const requireCompanyAdmin = (req, res, next) => {
try {
const attrs = req?.session?.attributes ?? {};
const ok = !!attrs.isCompanyAdmin || !!(req?.user?.isAdmin ?? attrs.isAdmin);
if (!ok) return res.status(403).json({ message: "Accès refusé - droits administrateur d'entreprise requis" });
next();
} catch (err) {
console.error("COMPANY ADMIN CHECK ERROR:", err);
return res.status(500).json({ message: "Erreur serveur" });
}
};

