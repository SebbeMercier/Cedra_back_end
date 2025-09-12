import { betterAuth } from "better-auth";
import { PostgresDialect } from "kysely";
import { Pool } from "pg";
import dotenv from "dotenv";

dotenv.config();

export const auth = betterAuth({
  database: {
    dialect: new PostgresDialect({
      pool: new Pool({
        connectionString: `postgresql://${process.env.DATABASE_USER}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:${process.env.DATABASE_PORT}/${process.env.DATABASE_NAME}`,
      }),
    }),
  },
  emailAndPassword: {
    enabled: true,
  },
  baseURL: process.env.BASE_URL,
});