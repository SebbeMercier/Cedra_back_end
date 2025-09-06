// controllers/companycontroller.js
import pool from "../config/db.js";

/**
 * GET /api/company/me
 * Retourne la société liée à l'utilisateur :
 *   1) via users.companyId si présent
 *   2) sinon société où il est admin (company_users.role='admin')
 *   3) sinon n'importe quelle société liée (employee)
 * Champs retournés : id, name, vat, billingStreet, billingPostalCode, billingCity, billingCountry
 */
export const meCompany = async (req, res) => {
  try {
    const userId = req.user.id;

    // 1) via users.companyId
    const [uRows] = await pool.query(
      "SELECT companyId FROM users WHERE id = ? LIMIT 1",
      [userId]
    );
    let companyId = uRows[0]?.companyId || null;

    // 2) sinon via company_users (priorité admin)
    if (!companyId) {
      const [cuAdmin] = await pool.query(
        `SELECT companyid AS companyId
         FROM company_users
         WHERE userid = ? AND LOWER(TRIM(role)) = 'admin'
         LIMIT 1`,
        [userId]
      );
      companyId = cuAdmin[0]?.companyId || null;
    }

    // 3) sinon n'importe quelle société liée
    if (!companyId) {
      const [cuAny] = await pool.query(
        `SELECT companyid AS companyId
         FROM company_users
         WHERE userid = ?
         LIMIT 1`,
        [userId]
      );
      companyId = cuAny[0]?.companyId || null;
    }

    if (!companyId) {
      return res.status(404).json({ message: "Aucune société liée" });
    }

    const [cRows] = await pool.query(
      `SELECT id, name, vat,
              billingStreet, billingPostalCode, billingCity, billingCountry
       FROM companies
       WHERE id = ?
       LIMIT 1`,
      [companyId]
    );

    if (cRows.length === 0) {
      return res.status(404).json({ message: "Société introuvable" });
    }

    res.json(cRows[0]);
  } catch (err) {
    console.error("ME COMPANY ERROR:", err);
    res.status(500).json({ message: "Erreur serveur" });
  }
};
