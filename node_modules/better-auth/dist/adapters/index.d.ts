import { b as AdapterConfig, C as CreateCustomAdapter, B as BetterAuthOptions, a as Adapter } from '../shared/better-auth.tThsKLej.js';
export { A as AdapterDebugLogs, e as AdapterTestDebugLogs, d as CleanedWhere, c as CustomAdapter } from '../shared/better-auth.tThsKLej.js';
import 'kysely';
import 'better-call';
import 'zod/v4';
import '../shared/better-auth.DTtXpZYr.js';
import '../shared/better-auth.B5FL4Q2B.js';
import 'zod/v4/core';
import 'zod';
import 'better-sqlite3';
import 'bun:sqlite';
import 'node:sqlite';

declare const createAdapter: ({ adapter, config: cfg, }: {
    config: AdapterConfig;
    adapter: CreateCustomAdapter;
}) => (options: BetterAuthOptions) => Adapter;

export { AdapterConfig, CreateCustomAdapter, createAdapter };
