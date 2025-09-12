import * as http from 'http';
import { IncomingHttpHeaders } from 'http';
import { g as Auth } from '../shared/better-auth.tThsKLej.js';
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

declare const toNodeHandler: (auth: {
    handler: Auth["handler"];
} | Auth["handler"]) => (req: http.IncomingMessage, res: http.ServerResponse) => Promise<void>;
declare function fromNodeHeaders(nodeHeaders: IncomingHttpHeaders): Headers;

export { fromNodeHeaders, toNodeHandler };
