import express from "express";
import nodemailer from "nodemailer";
import bcrypt from "bcryptjs";
import crypto from "crypto";
import pool from "../config/db.js";
import { authMiddleware } from "../middleware/authMiddleware.js";
import companyAdminMiddleware from "../middleware/companyAdminMiddleware.js";

const router = express.Router();

/* =========================
   SMTP
   ========================= */
function createTransport() {
  return nodemailer.createTransport({
    host: process.env.SMTP_HOST,
    port: Number(process.env.SMTP_PORT || 587),
    secure: Number(process.env.SMTP_PORT) === 465,
    auth: { user: process.env.SMTP_USER, pass: process.env.SMTP_PASS },
    logger: false,
    debug: false,
  });
}

function generatePassword(len = 12) {
  const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*";
  const bytes = crypto.randomBytes(len);
  let out = "";
  for (let i = 0; i < len; i++) out += alphabet[bytes[i] % alphabet.length];
  return out;
}

/* =========================
   Helpers
   ========================= */
async function getPrimaryCompanyForUser(userId) {
  const [rows] = await pool.query(
    `
    SELECT
      c.id, c.name,
      c.billingStreet, c.billingPostalCode, c.billingCity, c.billingCountry,
      cu.role,
      (LOWER(TRIM(cu.role)) = 'admin') AS isCompanyAdmin
    FROM company_users cu
    JOIN companies c ON c.id = cu.companyId
    WHERE cu.userId = ?
    ORDER BY (LOWER(TRIM(cu.role)) = 'admin') DESC, c.id ASC
    LIMIT 1
    `,
    [userId]
  );
  return rows[0] || null;
}

/* =========================
   GET /api/company/me
   -> société “principale” via company_users
   ========================= */
router.get("/me", authMiddleware, async (req, res) => {
  try {
    const company = await getPrimaryCompanyForUser(req.user.id);
    if (!company) return res.status(404).json({ message: "Aucune société" });
    return res.json({
      id: company.id,
      name: company.name,
      billingStreet: company.billingStreet,
      billingPostalCode: company.billingPostalCode,
      billingCity: company.billingCity,
      billingCountry: company.billingCountry,
      role: company.role,
      isCompanyAdmin: !!company.isCompanyAdmin,
    });
  } catch (e) {
    console.error("[COMPANY /me] ERROR:", e);
    return res.status(500).json({ message: "Erreur serveur" });
  }
});

/* =========================
   POST /api/company/invite
   ========================= */
router.post("/invite", authMiddleware, companyAdminMiddleware, async (req, res) => {
  const startedAt = Date.now();
  try {
    let { email, role } = req.body || {};
    if (!email) return res.status(400).json({ message: "Email requis" });

    const inviterCompany = await getPrimaryCompanyForUser(req.user.id);
    if (!inviterCompany) return res.status(403).json({ message: "Aucune société liée" });
    if (!inviterCompany.isCompanyAdmin) return res.status(403).json({ message: "Droits insuffisants" });
    const companyId = inviterCompany.id;

    const normalizedEmail = String(email).trim().toLowerCase();
    role = String(role || "employee").trim().toLowerCase();
    if (!["admin", "employee"].includes(role)) role = "employee";

    const displayName = normalizedEmail.split("@")[0];

    const conn = await pool.getConnection();
    let created = false;
    let tempPassword = null;
    let userId = null;

    try {
      await conn.beginTransaction();

      // 1) utilisateur existe ?
      const [u] = await conn.query(
        "SELECT id FROM users WHERE LOWER(email) = ? LIMIT 1",
        [normalizedEmail]
      );

      if (u.length === 0) {
        // 2) créer utilisateur local (⚠️ plus de users.companyId ici)
        tempPassword = generatePassword(12);
        const hash = await bcrypt.hash(tempPassword, 10);

        const [ins] = await conn.query(
          `INSERT INTO users (email, passwordHash, name, provider, is_admin)
           VALUES (?, ?, ?, 'local', 0)`,
          [normalizedEmail, hash, displayName]
        );
        userId = ins.insertId;
        created = true;
      } else {
        userId = u[0].id;
      }

      // 3) lier à la société (sans canOrder etc. si les colonnes n'existent pas)
      const [link] = await conn.query(
        "SELECT 1 FROM company_users WHERE companyId = ? AND userId = ? LIMIT 1",
        [companyId, userId]
      );

      if (link.length === 0) {
        await conn.query(
          "INSERT INTO company_users (companyId, userId, role) VALUES (?, ?, ?)",
          [companyId, userId, role]
        );
      } else {
        await conn.query(
          "UPDATE company_users SET role = ? WHERE companyId = ? AND userId = ?",
          [role, companyId, userId]
        );
      }

      await conn.commit();
    } catch (e) {
      await conn.rollback();
      throw e;
    } finally {
      conn.release();
    }

    // 4) e-mail (après commit)
    let emailSent = false, messageId = null, smtpResponse = null;
    try {
      const t = createTransport();
      const fromAddress = process.env.SMTP_FROM || process.env.SMTP_USER;

      const subject = created ? "Votre compte Cedra" : "Vous avez été ajouté à une société sur Cedra";
      const text = created
        ? `Bonjour,

Un compte Cedra a été créé pour vous par l’administrateur de votre société.

Identifiant : ${normalizedEmail}
Mot de passe temporaire : ${tempPassword}

Connectez-vous et changez votre mot de passe depuis votre profil.

L’équipe Cedra`
        : `Bonjour,

Vous avez été ajouté à la société de votre administrateur sur Cedra avec le rôle : ${role}.

Connectez-vous avec votre identifiant : ${normalizedEmail}.
Si vous avez oublié votre mot de passe, utilisez la fonction "Mot de passe oublié".

L’équipe Cedra`;

      const info = await t.sendMail({
        from: fromAddress,
        to: normalizedEmail,
        subject,
        text,
        envelope: { from: fromAddress, to: normalizedEmail },
      });

      emailSent = true;
      messageId = info.messageId;
      smtpResponse = info.response;
    } catch (mailErr) {
      console.error("[INVITE] MAIL SEND ERROR:", mailErr);
      // on n'échoue pas l'invite si le mail rate
    }

    const tookMs = Date.now() - startedAt;
    return res.status(created ? 201 : 200).json({
      message: created ? "Utilisateur créé et invité" : "Utilisateur existant associé à la société",
      created,
      userId,
      role,
      emailSent,
      messageId,
      smtpResponse,
      tookMs,
    });
  } catch (e) {
    console.error("[INVITE] ERROR:", e);
    return res.status(500).json({ message: "Erreur serveur" });
  }
});

/* =========================
   DEBUG SMTP
   ========================= */
router.get("/debug/smtp-verify", async (_req, res) => {
  try {
    const t = createTransport();
    await t.verify();
    res.setHeader("Content-Type", "application/json; charset=utf-8");
    res.status(200).send(JSON.stringify({ ok: true, message: "Connexion SMTP OK ✅" }));
  } catch (e) {
    console.error("SMTP VERIFY ERROR:", e);
    res.setHeader("Content-Type", "application/json; charset=utf-8");
    res.status(500).send(JSON.stringify({ ok: false, error: String(e) }));
  }
});

router.post("/debug/send-test", async (req, res) => {
  try {
    const to = (req.body?.to || process.env.SMTP_TEST_TO || "").trim();
    if (!to) return res.status(400).json({ ok: false, error: "Destinataire manquant (body.to ou SMTP_TEST_TO)" });

    const t = createTransport();
    const fromAddress = process.env.SMTP_FROM || process.env.SMTP_USER;
    const info = await t.sendMail({
      from: fromAddress,
      to,
      subject: "Test SMTP Cedra",
      text: "Ceci est un e-mail de test.",
      envelope: { from: fromAddress, to },
    });

    res.json({ ok: true, messageId: info.messageId, response: info.response });
  } catch (e) {
    console.error("SMTP SEND ERROR:", e);
    res.status(500).json({ ok: false, error: String(e) });
  }
});

/* =========================
   POST /api/company/users/:userId/reset-password
   -> génère un MDP temporaire + email
   ========================= */
router.post("/users/:userId/reset-password", authMiddleware, companyAdminMiddleware, async (req, res) => {
  try {
    const targetUserId = Number(req.params.userId);
    const company = await getPrimaryCompanyForUser(req.user.id);
    if (!company) return res.status(404).json({ message: "Aucune société" });

    // vérifie que le user appartient bien à la société
    const [chk] = await pool.query(
      "SELECT 1 FROM company_users WHERE companyId = ? AND userId = ? LIMIT 1",
      [company.id, targetUserId]
    );
    if (chk.length === 0) return res.status(404).json({ message: "Utilisateur non lié à cette société" });

    const tempPassword = generatePassword(12);
    const hash = await bcrypt.hash(tempPassword, 10);
    await pool.query("UPDATE users SET passwordHash = ? WHERE id = ?", [hash, targetUserId]);

    // email
    let emailSent = false, messageId = null, smtpResponse = null;
    try {
      const [u] = await pool.query("SELECT email FROM users WHERE id = ? LIMIT 1", [targetUserId]);
      const to = u[0]?.email;
      const t = createTransport();
      const fromAddress = process.env.SMTP_FROM || process.env.SMTP_USER;

      const info = await t.sendMail({
        from: fromAddress,
        to,
        subject: "Réinitialisation de votre mot de passe Cedra",
        text: `Bonjour,\n\nVotre mot de passe a été réinitialisé par un administrateur de la société ${company.name}.\n\nMot de passe temporaire : ${tempPassword}\n\nConnectez-vous et changez-le depuis votre profil.\n\nL’équipe Cedra`,
        envelope: { from: fromAddress, to },
      });

      emailSent = true;
      messageId = info.messageId;
      smtpResponse = info.response;
    } catch (mailErr) {
      console.error("[RESET PASSWORD MAIL ERROR]:", mailErr);
    }

    res.json({ ok: true, emailSent, messageId, smtpResponse });
  } catch (e) {
    console.error("[COMPANY POST /users/:id/reset-password] ERROR:", e);
    res.status(500).json({ message: "Erreur serveur" });
  }
});


export default router;
