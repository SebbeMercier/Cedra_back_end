import { A as AdapterDebugLogs, B as BetterAuthOptions, a as Adapter } from '../../shared/better-auth.tThsKLej.js';
import 'kysely';
import 'better-call';
import 'zod/v4';
import '../../shared/better-auth.DTtXpZYr.js';
import '../../shared/better-auth.B5FL4Q2B.js';
import 'zod/v4/core';
import 'zod';
import 'better-sqlite3';
import 'bun:sqlite';
import 'node:sqlite';

interface PrismaConfig {
    /**
     * Database provider.
     */
    provider: "sqlite" | "cockroachdb" | "mysql" | "postgresql" | "sqlserver" | "mongodb";
    /**
     * Enable debug logs for the adapter
     *
     * @default false
     */
    debugLogs?: AdapterDebugLogs;
    /**
     * Use plural table names
     *
     * @default false
     */
    usePlural?: boolean;
}
interface PrismaClient {
}
declare const prismaAdapter: (prisma: PrismaClient, config: PrismaConfig) => (options: BetterAuthOptions) => Adapter;

export { prismaAdapter };
export type { PrismaConfig };
