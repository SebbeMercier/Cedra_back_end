import { A as AdapterDebugLogs, B as BetterAuthOptions, a as Adapter } from '../../shared/better-auth.BNm-Id9Y.mjs';
import 'kysely';
import 'better-call';
import 'zod/v4';
import '../../shared/better-auth.DTtXpZYr.mjs';
import '../../shared/better-auth.DR57bygo.mjs';
import 'zod/v4/core';
import 'zod';
import 'better-sqlite3';
import 'bun:sqlite';
import 'node:sqlite';

interface MemoryDB {
    [key: string]: any[];
}
interface MemoryAdapterConfig {
    debugLogs?: AdapterDebugLogs;
}
declare const memoryAdapter: (db: MemoryDB, config?: MemoryAdapterConfig) => (options: BetterAuthOptions) => Adapter;

export { memoryAdapter };
export type { MemoryAdapterConfig, MemoryDB };
