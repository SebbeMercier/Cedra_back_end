// config/db.js
import pkg from "pg";
import dotenv from "dotenv";
dotenv.config();

const { Pool } = pkg;

// Préfère DATABASE_URL si définie, sinon DB_* (avec valeurs par défaut sûres)
const useUrl = !!process.env.DATABASE_URL;
const pool = useUrl
  ? new Pool({
      connectionString: process.env.DATABASE_URL, // ex: postgres://user:pass@192.168.1.83:5432/cedra_app
      // ssl: { rejectUnauthorized: false } // <- active si tu es en SSL
    })
  : new Pool({
      host: process.env.DB_HOST || "192.168.1.83",
      port: Number(process.env.DB_PORT || 5432),
      user: process.env.DB_USER || "cedra",
      password: process.env.DB_PASS || "cedra",
      database: process.env.DB_NAME || "cedra_app",
      // ssl: { rejectUnauthorized: false } // <- active si tu es en SSL
    });

pool.on("connect", () => {
  console.log(
    "✅ Connexion PostgreSQL OK ->",
    useUrl
      ? `DATABASE_URL (${new URL(process.env.DATABASE_URL).host})`
      : `${process.env.DB_HOST || "192.168.1.83"}:${process.env.DB_PORT || 5432}`
  );
});

pool.on("error", (err) => {
  console.error("❌ PG POOL ERROR:", err);
});

export default pool;
