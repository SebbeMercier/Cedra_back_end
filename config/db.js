import { betterAuth } from "better-auth";
import dotenv from "dotenv";
import { Pool } from "pg";

dotenv.config();

export const auth = betterAuth({
  database: {
    dialect: "postgresql",
    url: `postgresql://${process.env.DATABASE_USER}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:${process.env.DATABASE_PORT}/${process.env.DATABASE_NAME}`,
  },
  emailAndPassword: true,
  baseURL: process.env.BASE_URL,
});
