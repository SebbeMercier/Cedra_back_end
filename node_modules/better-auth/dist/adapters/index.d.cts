import { b as AdapterConfig, C as CreateCustomAdapter, B as BetterAuthOptions, a as Adapter } from '../shared/better-auth.BWDlmABj.cjs';
export { A as AdapterDebugLogs, e as AdapterTestDebugLogs, d as CleanedWhere, c as CustomAdapter } from '../shared/better-auth.BWDlmABj.cjs';
import 'kysely';
import 'better-call';
import 'zod/v4';
import '../shared/better-auth.DTtXpZYr.cjs';
import '../shared/better-auth.BaMSx6K3.cjs';
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
