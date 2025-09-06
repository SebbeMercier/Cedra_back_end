// config/lucia.js
import { Lucia } from "lucia";
import { NodePostgresAdapter } from "@lucia-auth/adapter-postgresql"; // ✅ adapter officiel pour 'pg'
import pool from "./db.js";

const adapter = new NodePostgresAdapter(pool, {
  user: "users",
  key: "user_keys",          // <- ta table des clés (email/password, oauth, etc.)
  session: "user_sessions",  // <- ta table des sessions
});

export const auth = new Lucia(adapter, {
  env: process.env.NODE_ENV === "production" ? "PROD" : "DEV",
  sessionCookie: {
    expires: false,
    attributes: {
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
    },
  },
  getUserAttributes: (d) => ({
    email: d.email,
    name: d.name,
    isAdmin: d.is_admin,
    isSuspended: d.is_suspended,
    provider: d.provider,
  }),
});
