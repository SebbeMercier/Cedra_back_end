// controllers/addressescontroller.js
import { pool } from "../config/db.js";

/**
 * GET /api/addresses/mine
 * Retourne :
 * - tes adresses perso (user_id = req.user.id)
 * - ET les adresses d’entreprise des sociétés auxquelles tu es rattaché
 *   • partagées (user_id IS NULL)
 *   • OU privées à toi (user_id = req.user.id)
 *
 * Le champ `type` est CALCULÉ (pas de colonne en DB) :
 *  - 'user'    si user_id != null et companyId == null
 *  - 'company' si user_id == null et companyId != null
 *  - 'both'    si user_id != null et companyId != null (rare)
 *  - 'unknown' sinon
 */
export const listMine = async (req, res) => {
  try {
    const userId = req.user.id;

    const [rows] = await pool.query(
      `
      SELECT
        a.id,
        a.street,
        a.postalCode                AS postalCode,
        a.city,
        a.country,
        a.isDefault                 AS isDefault,
        a.user_id                   AS userId,
        a.companyId                 AS companyId,
        CASE
          WHEN a.user_id IS NOT NULL AND a.companyId IS NOT NULL THEN 'both'
          WHEN a.user_id IS NOT NULL THEN 'user'
          WHEN a.companyId IS NOT NULL THEN 'company'
          ELSE 'unknown'
        END                         AS type
      FROM addresses a
      WHERE a.user_id = ?
         OR (
              a.companyId IN (
                SELECT cu.companyid
                FROM company_users cu
                WHERE cu.userid = ?
              )
              AND (a.user_id IS NULL OR a.user_id = ?)
            )
      ORDER BY a.id DESC
      `,
      [userId, userId, userId]
    );

    res.json(rows);
  } catch (err) {
    console.error("ADDRESSES MINE ERROR:", err);
    res.status(500).json({ message: "Erreur serveur" });
  }
};

/**
 * POST /api/addresses
 * Body: { street, postalCode, city, country, type: 'user'|'company', companyId?, privateCompany? }
 *
 * - type = 'user'    → adresse perso (user_id = req.user.id, companyId = NULL)
 * - type = 'company' → adresse d’entreprise :
 *        privateCompany=true (défaut)  → privée à l’utilisateur (user_id = req.user.id)
 *        privateCompany=false          → partagée (user_id = NULL)
 */
export const createAddress = async (req, res) => {
  try {
    const userId = req.user.id;
    const {
      street,
      postalCode,
      city,
      country,
      type = "user",
      companyId,
      privateCompany = true,
    } = req.body;

    if (!street || !postalCode || !city || !country) {
      return res.status(400).json({ message: "Champs d'adresse incomplets" });
    }

    if (type === "company") {
      if (!companyId) {
        return res.status(400).json({ message: "companyId requis pour type=company" });
      }

      // Vérifie l’appartenance à la société (admin OU employé)
      const [r] = await pool.query(
        `SELECT 1 FROM company_users WHERE userid = ? AND companyid = ? LIMIT 1`,
        [userId, companyId]
      );
      if (r.length === 0) {
        return res.status(403).json({ message: "Non autorisé pour cette société" });
      }

      const userIdValue = privateCompany ? userId : null;

      const [ins] = await pool.query(
        `INSERT INTO addresses (street, postalCode, city, country, user_id, companyId, isDefault)
         VALUES (?, ?, ?, ?, ?, ?, 0)`,
        [street, postalCode, city, country, userIdValue, companyId]
      );

      return res.status(201).json({ id: ins.insertId });
    }

    // Par défaut: adresse perso
    const [ins] = await pool.query(
      `INSERT INTO addresses (street, postalCode, city, country, user_id, companyId, isDefault)
       VALUES (?, ?, ?, ?, ?, NULL, 0)`,
      [street, postalCode, city, country, userId]
    );

    res.status(201).json({ id: ins.insertId });
  } catch (err) {
    console.error("ADDRESS CREATE ERROR:", err);
    res.status(500).json({ message: "Erreur serveur" });
  }
};
