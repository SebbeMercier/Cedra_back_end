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

interface MemoryDB {
    [key: string]: any[];
}
interface MemoryAdapterConfig {
    debugLogs?: AdapterDebugLogs;
}
declare const memoryAdapter: (db: MemoryDB, config?: MemoryAdapterConfig) => (options: BetterAuthOptions) => Adapter;

export { memoryAdapter };
export type { MemoryAdapterConfig, MemoryDB };
